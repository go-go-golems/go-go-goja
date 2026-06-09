// Package hotreload provides a small blue/green runtime manager for xgoja
// embedding applications.
//
// The manager owns the currently active gojahttp host/runtime snapshot. A reload
// builds a fresh candidate host, lets the application create and bootstrap a new
// runtime against that host, optionally smoke-tests the candidate, and then
// atomically swaps it into service. Failed reloads leave the previous snapshot
// active.
//
// Generated xgoja package hosts usually use Candidate.Host with the HTTP
// provider's external host service:
//
//	bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
//		ConfigureServices: func(services *app.HostServices) {
//			_ = services.SetHostService(httpprovider.HostServiceKey, httpprovider.ExternalHostService{
//				Host:       candidate.Host,
//				OwnsListen: false,
//			})
//		},
//	})
//
// The outer Go application mounts Manager on its own mux and owns the listener.
package hotreload
