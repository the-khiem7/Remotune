// This module is deliberately the only frontend location that imports generated
// Wails bindings. The generator owns the actual files under frontend/bindings.
export { GetAutostartStatus, GetProfileSettings, Pause, PortablePathStatus, RestoreNow, Resume, SetAutostart, SetProfileSettings, Status, VisualEffectNames } from '../bindings/github.com/khiemnguyen/remotune/internal/lifecycle/service'
