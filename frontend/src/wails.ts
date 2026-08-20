// This module is deliberately the only frontend location that imports generated
// Wails bindings. The generator owns the actual files under frontend/bindings.
export { GetAutostartStatus, Pause, PortablePathStatus, RestoreNow, Resume, SetAutostart, Status } from '../bindings/github.com/khiemnguyen/remotune/internal/lifecycle/service'
