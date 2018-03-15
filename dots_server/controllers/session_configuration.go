package controllers

import (
	"fmt"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"strings"
	"strconv"

	common "github.com/nttdots/go-dots/dots_common"
	"github.com/nttdots/go-dots/dots_common/messages"
	"github.com/nttdots/go-dots/dots_server/models"
	dots_config "github.com/nttdots/go-dots/dots_server/config"
)

/*
 * Controller for the session_configuration API.
 */
type SessionConfiguration struct {
	Controller
}

func (m *SessionConfiguration) HandleGet(request Request, customer *models.Customer) (res Response, err error) {
	
	log.WithField("request", request).Debug("[GET] receive message")

	sid, err := parseSidFromUriPath(request.PathInfo)
	if err != nil {
		log.Errorf("Failed to parse Uri-Path, error: %s", err)
		res = Response{
			Type: common.Acknowledgement,
			Code: common.BadRequest,
			Body: nil,
		}
		return
	}

	// TODO: check found or not

	config := dots_config.GetServerSystemConfig().SignalConfigurationParameter

	resp := messages.ConfigurationResponse{}
	resp.SignalConfigs = messages.ConfigurationResponseConfigs{}
	resp.SignalConfigs.MitigationConfig = messages.ConfigurationResponseConfig{}
	resp.SignalConfigs.MitigationConfig.HeartbeatInterval.SetMinMax(config.HeartbeatInterval)
	resp.SignalConfigs.MitigationConfig.MissingHbAllowed.SetMinMax(config.MissingHbAllowed)
	resp.SignalConfigs.MitigationConfig.MaxRetransmit.SetMinMax(config.MaxRetransmit)
	resp.SignalConfigs.MitigationConfig.AckTimeout.SetMinMax(config.AckTimeout)
	resp.SignalConfigs.MitigationConfig.AckRandomFactor.SetMinMax(config.AckRandomFactor)

	log.Debugf("%s", sid)

	if sid != 0 {
		signalSessionConfiguration, err := models.GetSignalSessionConfiguration(customer.Id, sid)
		if err != nil {
			res = Response{
				Type: common.Acknowledgement,
				Code: common.NotFound,
				Body: nil,
			}
			return res, err
		}
		resp.SignalConfigs.SessionId = sid
		resp.SignalConfigs.MitigationConfig.HeartbeatInterval.CurrentValue = signalSessionConfiguration.HeartbeatInterval
		resp.SignalConfigs.MitigationConfig.MissingHbAllowed.CurrentValue  = signalSessionConfiguration.MissingHbAllowed
		resp.SignalConfigs.MitigationConfig.MaxRetransmit.CurrentValue     = signalSessionConfiguration.MaxRetransmit
		resp.SignalConfigs.MitigationConfig.AckTimeout.CurrentValue        = signalSessionConfiguration.AckTimeout
		resp.SignalConfigs.MitigationConfig.AckRandomFactor.CurrentValue   = decimal.NewFromFloat(signalSessionConfiguration.AckRandomFactor)
		resp.SignalConfigs.MitigationConfig.TriggerMitigation              = signalSessionConfiguration.TriggerMitigation
	} else {
		defaultValue := dots_config.GetServerSystemConfig().DefaultSignalConfiguration

		resp.SignalConfigs.MitigationConfig.HeartbeatInterval.CurrentValue = defaultValue.HeartbeatInterval
		resp.SignalConfigs.MitigationConfig.MissingHbAllowed.CurrentValue  = defaultValue.MissingHbAllowed
		resp.SignalConfigs.MitigationConfig.MaxRetransmit.CurrentValue     = defaultValue.MaxRetransmit
		resp.SignalConfigs.MitigationConfig.AckTimeout.CurrentValue        = defaultValue.AckTimeout
		resp.SignalConfigs.MitigationConfig.AckRandomFactor.CurrentValue   = decimal.NewFromFloat(defaultValue.AckRandomFactor).Round(2)
		resp.SignalConfigs.MitigationConfig.TriggerMitigation              = true
	}

	// TODO: support Idle-Config
	res = Response{
			Type: common.Acknowledgement,
			Code: common.Content,
			Body: resp,
	}

	return
}

/*
 * Handles session_configuration PUT requests and start the mitigation.
 *  1. Validate the received session configuration requests.
 *  2. return the validation results.
 *
 * parameter:
 *  request request message
 *  customer request source Customer
 * return:
 *  res response message
 *  err error
 */
func (m *SessionConfiguration) HandlePut(newRequest Request, customer *models.Customer) (res Response, err error) {

	request := newRequest.Body

	if request == nil {
		res = Response{
			Type: common.Acknowledgement,
			Code: common.BadRequest,
			Body: nil,
		}
		return
	}

	payload := &request.(*messages.SignalConfigRequest).SignalConfigs.MitigationConfig
	sessionConfigurationPayloadDisplay(payload)
	// TODO: support IdleConfig, draft-17+

	ackRandomFactor, _ := payload.AckRandomFactor.CurrentValue.Float64()
	// validate
	signalSessionConfiguration := models.NewSignalSessionConfiguration(
		payload.SessionId,
		payload.HeartbeatInterval.CurrentValue,
		payload.MissingHbAllowed.CurrentValue,
		payload.MaxRetransmit.CurrentValue,
		payload.AckTimeout.CurrentValue,
		ackRandomFactor,
		payload.TriggerMitigation,
	)
	v := models.SignalConfigurationValidator{}
	validateResult := v.Validate(signalSessionConfiguration, *customer)
	if !validateResult {
		goto ResponseNG
	} else {
		// Register SignalConfigurationParameter
		_, err = models.CreateSignalSessionConfiguration(*signalSessionConfiguration, *customer)
		if err != nil {
			goto ResponseNG
		}

		goto ResponseOK
	}

ResponseNG:
// on validation error
	res = Response{
		Type: common.Acknowledgement,
		Code: common.BadRequest,
		Body: nil,
	}
	return
ResponseOK:
// on validation success
	res = Response{
		Type: common.Acknowledgement,
		Code: common.Created,
		Body: nil,
	}
	return
}

func (m *SessionConfiguration) HandleDelete(newRequest Request, customer *models.Customer) (res Response, err error) {

	log.WithField("request", newRequest).Debug("[DELETE] receive message")

	sid, err := parseSidFromUriPath(newRequest.PathInfo)
	if err != nil {
		log.Errorf("Failed to parse Uri-Path, error: %s", err)
		res = Response{
			Type: common.Acknowledgement,
			Code: common.BadRequest,
			Body: nil,
		}
		return
	}

	if sid != 0{
		err = models.DeleteSignalSessionConfiguration(customer.Id, sid)
	} else {
		err = models.DeleteSignalSessionConfigurationByCustomerId(customer.Id)
	}

	if err != nil {
		res = Response{
			Type: common.Acknowledgement,
			Code: common.InternalServerError,
			Body: nil,
		}
		return
	}

	res = Response{
		Type: common.Acknowledgement,
		Code: common.Deleted,
		Body: nil,
	}
	return
}


/*
 * Parse the request body and display the contents of the messages to stdout.
*/
func sessionConfigurationPayloadDisplay(data *messages.SignalConfig) {

	var result string = "\n"
	result += fmt.Sprintf("   \"%s\": %d\n", "session-id", data.SessionId)
	result += fmt.Sprintf("   \"%s\": %d\n", "heartbeat-interval", data.HeartbeatInterval)
	result += fmt.Sprintf("   \"%s\": %d\n", "missing-hb-allowed", data.MissingHbAllowed)
	result += fmt.Sprintf("   \"%s\": %d\n", "max-retransmit", data.MaxRetransmit)
	result += fmt.Sprintf("   \"%s\": %d\n", "ack-timeout", data.AckTimeout)
	result += fmt.Sprintf("   \"%s\": %f\n", "ack-random-factor", data.AckRandomFactor)
	result += fmt.Sprintf("   \"%s\": %f\n", "trigger-mitigation", data.TriggerMitigation)
	log.Infoln(result)
}

/*
*  Get sid value from URI-Path
*/
func parseSidFromUriPath(uriPath []string) (sid int, err error){
	log.Debugf("Parsing URI-Path : %+v", uriPath)
	// Get sid from Uri-Path
	for _, uriPath := range uriPath{
		if(strings.HasPrefix(uriPath, "sid")){
			sidStr := uriPath[strings.Index(uriPath, "=")+1:]
			sidValue, err := strconv.Atoi(sidStr)
			if err != nil {
				log.Errorf("Mid is not integer type.")
				return sid, err
			}
			sid = sidValue
		}
	}
	log.Debugf("Parsing URI-Path result : sid=%+v", sid)
	return
}