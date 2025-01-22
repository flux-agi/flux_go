package flux

import "fmt"

type Topics struct {
	service string
}

func NewTopics(service string) *Topics {
	return &Topics{service: service}
}

func (t *Topics) GetConfig() string                      { return fmt.Sprintf("service/%s/get_config", t.service) }
func (t *Topics) SetConfig() string                      { return fmt.Sprintf("service/%s/set_config", t.service) }
func (t *Topics) SendStatus() string                     { return fmt.Sprintf("service/%s/status", t.service) }
func (t *Topics) RequestStatus() string                  { return fmt.Sprintf("service/%s/request_status", t.service) }
func (t *Topics) Start() string                          { return fmt.Sprintf("service/%s/start", t.service) }
func (t *Topics) Stop() string                           { return fmt.Sprintf("service/%s/stop", t.service) }
func (t *Topics) Restart() string                        { return fmt.Sprintf("service/%s/restart", t.service) }
func (t *Topics) PushDevelopmentMode(guid string) string { return "service/development_mode/" + guid }
func (t *Topics) Errors() string                         { return "service/*/error" }
func (t *Topics) GlobalTick() string                     { return "service/tick" }
func (t *Topics) GetCommonState() string {
	return fmt.Sprintf("service/%s/get_common_state", t.service)
}
func (t *Topics) SetCommonState() string {
	return fmt.Sprintf("service/%s/set_common_state", t.service)
}
