# killswitch — context

**Purpose:** Decide whether the agent must stop claiming/executing jobs.

**Public surface:** `CloudFlag` interface (`MachineKillSwitch(ctx, machineID) (bool, error)`),
`Checker`, `New(cloud, machineID, sentinel)`, `(*Checker).Tripped(ctx) (bool, error)`.

**Design / flow:** `Tripped` checks the local sentinel file first — if it exists, the
switch is engaged and the cloud is not consulted (a disconnected machine can still be
hard-stopped). Otherwise it reads the cloud flag via `CloudFlag` (the `store.Store`
implements this with `MachineKillSwitch`). The `CloudFlag` interface keeps this package
decoupled from the store and unit-testable with a fake.

**Depends on:** stdlib (`os`, `io/fs`); a `CloudFlag` implementation at the call site.

**Extending it:** keep the local-precedence ordering; if more stop sources are added
(e.g. a hardware signal), check them before the cloud round-trip.
