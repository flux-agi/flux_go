package flux

func (n *Service[T]) OnNodeReady(handler func(cfg NodeConfig[T]) error) error {
	n.nodeHandlers.OnReady(handler)
	return nil
}

func (n *Service[T]) OnNodeStart(handler NodeEventHandler) {
	n.nodeHandlers.OnStart(handler)
}

func (n *Service[T]) OnNodeStop(handler NodeEventHandler) {
	n.nodeHandlers.OnStop(handler)
}

func (n *Service[T]) OnNodeSubscribe(port string, handler func(node NodeConfig[T], payload []byte) error) error {
	// todo: need implement
	return nil
}
