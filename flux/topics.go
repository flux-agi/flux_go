package flux

import "fmt"

type ServiceTopics struct {
	service string
}

func NewTopics(service string) *ServiceTopics {
	return &ServiceTopics{service: service}
}

func (t *ServiceTopics) RequestConfig() string {
	return fmt.Sprintf("service.%s.get_config", t.service)
}
func (t *ServiceTopics) ResponseConfig() string {
	return fmt.Sprintf("service.%s.set_config", t.service)
}
func (t *ServiceTopics) SendStatus() string { return fmt.Sprintf("service.%s.status", t.service) }
func (t *ServiceTopics) RequestStatus() string {
	return fmt.Sprintf("service.%s.request_status", t.service)
}
func (t *ServiceTopics) Start() string   { return fmt.Sprintf("service.%s.start", t.service) }
func (t *ServiceTopics) Stop() string    { return fmt.Sprintf("service.%s.stop", t.service) }
func (t *ServiceTopics) Restart() string { return fmt.Sprintf("service.%s.restart", t.service) }
func (t *ServiceTopics) PushDevelopmentMode(guid string) string {
	return "service.development_mode." + guid
}
func (t *ServiceTopics) Errors() string     { return "service." + ".error" }
func (t *ServiceTopics) GlobalTick() string { return "service.tick" }
func (t *ServiceTopics) GetCommonState() string {
	return fmt.Sprintf("service.%s.get_common_state", t.service)
}
func (t *ServiceTopics) SetCommonState() string {
	return fmt.Sprintf("service.%s.set_common_state", t.service)
}
func (t *ServiceTopics) GetCommonData() string {
	return fmt.Sprintf("service.%s.get_common_data", t.service)
}

func (t *ServiceTopics) SetCommonData() string {
	return fmt.Sprintf("service.%s.set_common_data", t.service)
}

// IDEStatus returns topic for subscribing on IDE statuses.
//
// When client connects to the manager, manager sends "status": "CONNECTED" into this topic.
// When client disconnected, manager sends "status": "DISCONNECTED" into this topic.
func (t *ServiceTopics) IDEStatus() string {
	return "ide.status"
}
