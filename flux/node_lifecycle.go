package flux

import "time"

func (s *Service[T]) OnNodeReady(handler func(cfg NodeConfig[T]) error) {
	s.nodeHandlers.OnReady(handler)
}

func (s *Service[T]) OnNodeStart(handler NodeEventHandler) {
	s.nodeHandlers.OnStart(handler)
}

func (s *Service[T]) OnNodeStop(handler NodeEventHandler) {
	s.nodeHandlers.OnStop(handler)
}

func (s *Service[T]) OnNodeDestroy(handler func(node NodeConfig[T]) error) {
	s.nodeHandlers.OnDestroy(handler)
}

func (s *Service[T]) OnNodeSubscribe(port string, handler func(node NodeConfig[T], payload []byte) error) error {
	s.nodeHandlers.OnSubscribe(port, handler)
	for _, node := range s.nodes {
		if err := node.OnSubscribe(port, handler); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service[T]) OnNodeTick(handler func(node NodeConfig[T], deltaTime time.Duration, timestamp time.Time) error) {
	s.nodeHandlers.OnTick(handler)
}
