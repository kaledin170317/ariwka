package swaggerdocs

import _ "embed"

//go:embed swagger/control/control.swagger.json
var Control []byte

//go:embed swagger/node/node.swagger.json
var Node []byte
