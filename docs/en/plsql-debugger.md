# PL/SQL debugger: design status

[한국어 설계 기록](../plsql-debugger.md) | **English summary** · [Documentation](index.md)

This is a summary of the **2026-10-22 LORA-53 feasibility/design record**, not an implemented feature. **v0.10.2 has no breakpoint or step debugger.** Source viewing/editing/compilation, argument inspection and routine execution are implemented separately and do not imply debugger support.

The dated recommendation is to prove a minimal JDWP connection against a modern Oracle test server before deciding on a TUI debugger MVP. Oracle 10g requires separate server proof. The original review compares DBMS_DEBUG and DBMS_DEBUG_JDWP and links Oracle's version-specific APIs; it does not establish a working go-ora debugger integration.

A possible design uses a dedicated target workspace/session plus a separate JDWP control/receiver channel. Reusing the single SQL work queue for debugger controls could deadlock while target execution is paused. The database must reach the debugger's TCP endpoint; a working SQL connection alone does not prove the reverse network path. Rights, network ACLs, debug compilation, Oracle extensions, source mapping, value types, cancellation and cleanup need explicit proof.

Automatic broad privilege grants, recompiling existing objects and copying session transaction/state are not part of implemented behavior. Debug compilation is DDL and needs its own transaction warning/decision. Proposed debugger-specific keys apply only to a future debugger view; they do not alter current global keymaps.

Read [the original design and primary-source links](../plsql-debugger.md) for the version/date-specific assumptions, proof gates and deferred work. Do not treat the proposal's API table or size estimates as a current product guarantee.
