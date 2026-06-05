// Package host exposes explicitly guarded host-capability modules as an xgoja
// provider package.
//
// The provider package ID is "go-go-goja-host". It registers fs, node:fs,
// exec, database, and db. These modules can read/write host files, execute
// host processes, or connect to databases. They are intentionally separate
// from the core provider package and require explicit runtime-profile config.
//
// Security model:
//   - fs/node:fs require config.allow=true for host filesystem access. This is
//     an acknowledgement gate; the underlying module is not path-sandboxed.
//   - fs/node:fs can alternatively use config.embedded.allow=true with mounts
//     to expose read-only embedded assets. Do not combine host allow=true and
//     embedded.allow=true in one module instance; register separate aliases such
//     as fs:host and fs:assets.
//   - exec requires config.allow=true. allowedCommands can restrict exact
//     command names; if omitted, any command is allowed.
//   - database/db disable JavaScript configure() by default. Set
//     config.allowConfigure=true to allow scripts to open driver/data-source
//     pairs such as sqlite3 in-memory databases.
//   - database/db can be preconfigured by the generated binary with
//     config.driverName and config.dataSourceName. In preconfigured mode,
//     JavaScript can call query/exec/begin/close immediately and configure() is
//     rejected even if allowConfigure is also set.
//
// Example preconfigured database module instance:
//
//	modules:
//	  - package: go-go-goja-host
//	    name: db
//	    as: db
//	    config:
//	      driverName: sqlite3
//	      dataSourceName: ':memory:'
//
// Example script-owned demo configuration:
//
//	modules:
//	  - package: go-go-goja-host
//	    name: db
//	    as: db
//	    config:
//	      allowConfigure: true
package host
