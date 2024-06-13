// This file is generated by "./lib/proto/generate"

package proto

/*

Extensions

Defines commands and events for browser extensions. Available if the client
is connected using the --remote-debugging-pipe flag and
the --enable-unsafe-extension-debugging flag is set.

*/

// ExtensionsLoadUnpacked Installs an unpacked extension from the filesystem similar to
// --load-extension CLI flags. Returns extension ID once the extension
// has been installed.
type ExtensionsLoadUnpacked struct {
	// Path Absolute file path.
	Path string `json:"path"`
}

// ProtoReq name.
func (m ExtensionsLoadUnpacked) ProtoReq() string { return "Extensions.loadUnpacked" }

// Call the request.
func (m ExtensionsLoadUnpacked) Call(c Client) (*ExtensionsLoadUnpackedResult, error) {
	var res ExtensionsLoadUnpackedResult
	return &res, call(m.ProtoReq(), m, &res, c)
}

// ExtensionsLoadUnpackedResult ...
type ExtensionsLoadUnpackedResult struct {
	// ID Extension id.
	ID string `json:"id"`
}