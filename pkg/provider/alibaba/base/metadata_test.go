package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ctrlCfg "k8s.io/alibaba-load-balancer-controller/pkg/config"
)

func TestVswitchID(t *testing.T) {
	raw := ctrlCfg.CloudCFG.Global.VswitchID
	cfg := &CfgMetaData{base: NewBaseMetaData(nil)}

	ctrlCfg.CloudCFG.Global.VswitchID = "vsw-a"
	vsw, err := cfg.VswitchID()
	assert.Equal(t, nil, err)
	assert.Equal(t, "vsw-a", vsw)

	ctrlCfg.CloudCFG.Global.VswitchID = ":vsw-b"
	vsw, err = cfg.VswitchID()
	assert.Equal(t, nil, err)
	assert.Equal(t, "vsw-b", vsw)

	ctrlCfg.CloudCFG.Global.VswitchID = ":vsw-c,:vsw-d"
	vsw, err = cfg.VswitchID()
	assert.Equal(t, nil, err)
	assert.Equal(t, "vsw-c", vsw)

	ctrlCfg.CloudCFG.Global.VswitchID = "cn-hangzhou-h:vsw-h"
	vsw, err = cfg.VswitchID()
	assert.Equal(t, nil, err)
	assert.Equal(t, "vsw-h", vsw)

	ctrlCfg.CloudCFG.Global.VswitchID = "cn-hangzhou-h:vsw-e,cn-hangzhou-f:vsw-f"
	vsw, err = cfg.VswitchID()
	assert.Equal(t, nil, err)
	assert.Equal(t, "vsw-e", vsw)

	ctrlCfg.CloudCFG.Global.VswitchID = raw
}
