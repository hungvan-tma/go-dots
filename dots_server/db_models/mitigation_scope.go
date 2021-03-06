package db_models

import "time"

type MitigationScope struct {
	Id               int64     `xorm:"'id' pk autoincr"`
	CustomerId       int       `xorm:"'customer_id'"`
	ClientIdentifier string    `xorm:"'client_identifier'"`
	ClientDomainIdentifier string `xorm:"'client_domain_identifier'"`
	MitigationId     int       `xorm:"'mitigation_id'"`
	Status			 int	   `xorm:"'status'"`
	Lifetime         int       `xorm:"'lifetime'"`
	AttackStatus	 int	   `xorm:"'attack-status'"`
	Created          time.Time `xorm:"created"`
	Updated          time.Time `xorm:"updated"`
}
