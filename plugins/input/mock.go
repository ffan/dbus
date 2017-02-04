package input

import (
	"github.com/funkygao/dbus/engine"
	"github.com/funkygao/dbus/plugins/model"
	conf "github.com/funkygao/jsconf"
)

type MockInput struct {
	stopChan chan struct{}
}

func (this *MockInput) Init(config *conf.Conf) {
	this.stopChan = make(chan struct{})
}

func (this *MockInput) Run(r engine.InputRunner, h engine.PluginHelper) error {
	for {
		select {
		case <-this.stopChan:
			return nil

		case pack, ok := <-r.InChan():
			if !ok {
				break
			}

			pack.Payload = model.Bytes("hello world")
			r.Inject(pack)
		}
	}

	return nil
}

func (this *MockInput) Stop() {
	close(this.stopChan)
}

func init() {
	engine.RegisterPlugin("MockInput", func() engine.Plugin {
		return new(MockInput)
	})
}
