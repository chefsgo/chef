package chef

import (
	. "github.com/chefsgo/base"
)

var (
	mBridge = &bridgeModule{}
)

type (
	bridgeModule struct {
		token tokenBridge
	}

	tokenBridge interface {
		Validate(token string) error
	}
)

func (this *bridgeModule) Register(name string, value Any, override bool) {
	// switch obj := value.(type) {
	// case tokenBridge:
	// 	this.token = value
	// }
}
func (this *bridgeModule) Configure(config Map) {
}
func (this *bridgeModule) Initialize() {
}
func (this *bridgeModule) Connect() {
}
func (this *bridgeModule) Launch() {
}
func (this *bridgeModule) Terminate() {
}
