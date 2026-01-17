package pubsub

func (g *Global) GetMap(channel string) map[*Connection]struct{} {
	g.Mu.RLock()
	defer g.Mu.RUnlock()
	cons, ok := g.ChannelToClient[channel]
	if ok {
		return cons
	}
	return nil
}

func (g *Global) DeliverMessage(cons map[*Connection]struct{}, payload []byte) {
	if cons == nil {
		return
	}

	// Copy connections under read lock
	conns := make([]*Connection, 0, len(cons))

	g.Mu.RLock()
	for c := range cons {
		conns = append(conns, c)
	}
	g.Mu.RUnlock()

	// Write to connections without holding the global lock
	for _, conn := range conns {
		conn.Mu.Lock()
		conn.W.Write(payload)
		conn.W.Flush()
		conn.Mu.Unlock()
	}
}
