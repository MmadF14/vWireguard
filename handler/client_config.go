package handler

import (
	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
	"github.com/MmadF14/vwireguard/util"
)

// clientConfig builds a client's config against the interface it lives on, so
// clients on wg1 / an AmneziaWG interface get that interface's key, port and
// obfuscation parameters instead of the default server's.
func clientConfig(db store.IStore, client model.Client, server model.Server, settings model.GlobalSetting) string {
	return util.BuildClientConfigForInterface(client, server, settings,
		util.ClientInterface(&client, settings, db.GetInterface))
}
