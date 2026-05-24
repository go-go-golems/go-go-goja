// Package host exposes explicitly guarded host-capability modules as an xgoja
// provider package.
//
// The provider package ID is "go-go-goja-host". It registers fs, node:fs,
// exec, database, and db. These modules can read/write host files, execute
// host processes, or connect to databases. They are intentionally separate
// from the core provider package and require explicit runtime-profile config.
//
// Security model:
//   - fs/node:fs require config.allow=true. This is an acknowledgement gate;
//     the underlying module is not path-sandboxed.
//   - exec requires config.allow=true. allowedCommands can restrict exact
//     command names; if omitted, any command is allowed.
//   - database/db disable JavaScript configure() by default. Set
//     config.allowConfigure=true to allow scripts to open driver/data-source
//     pairs such as sqlite3 in-memory databases.
package host
