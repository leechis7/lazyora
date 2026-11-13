# PL/SQL debugger: design status

[한국어 설계 기록](../plsql-debugger.md) | **English summary** · [Documentation](index.md)

## LORA-88 implementation status

**Execute → PL/SQL debugger** runs a standalone parameterless procedure owned by the login user. Ask the DBA for `DEBUG CONNECT SESSION` and compile the target beforehand with `ALTER PROCEDURE name COMPILE DEBUG`. Compilation is DDL; use a connection with no pending work.

Enter an Oracle source line to set the first breakpoint. **F11** continues, **F8** steps over, **F7** steps into, **Alt+O** returns to the caller, and **Esc** stops or closes. While paused, use **Alt+B** to add a breakpoint to the displayed procedure or **Alt+W** to read a scalar by name in the current frame. The source marks the current line with `>`. Use **Alt+V** to focus the source and **Tab** to move to the call stack and controls.

The first version was tested on Oracle Free 23. It uses two temporary Oracle sessions, separate from the editor transaction, with no listening port. Uncommitted work on the debug connection is rolled back at the end. COMMIT, DDL, and autonomous transactions in the procedure can still have persistent effects. Oracle 10g, 19c and RAC have not been tested live. Starting a function or package directly and editing parameters or variables are unsupported.

The remainder is the original LORA-53 design record.


This is a summary of the **2026-10-22 LORA-53 feasibility/design record**, not an implemented feature. **v0.10.2 has no breakpoint or step debugger.** Source viewing/editing/compilation, argument inspection and routine execution are implemented separately and do not imply debugger support.

The dated recommendation is to prove a minimal JDWP connection against a modern Oracle test server before deciding on a TUI debugger MVP. Oracle 10g requires separate server proof. The original review compares DBMS_DEBUG and DBMS_DEBUG_JDWP and links Oracle's version-specific APIs; it does not establish a working go-ora debugger integration.

A possible design uses a dedicated target workspace/session plus a separate JDWP control/receiver channel. Reusing the single SQL work queue for debugger controls could deadlock while target execution is paused. The database must reach the debugger's TCP endpoint; a working SQL connection alone does not prove the reverse network path. Rights, network ACLs, debug compilation, Oracle extensions, source mapping, value types, cancellation and cleanup need explicit proof.

Automatic broad privilege grants, recompiling existing objects and copying session transaction/state are not part of implemented behavior. Debug compilation is DDL and needs its own transaction warning/decision. Proposed debugger-specific keys apply only to a future debugger view; they do not alter current global keymaps.

Read [the original design and primary-source links](../plsql-debugger.md) for the version/date-specific assumptions, proof gates and deferred work. Do not treat the proposal's API table or size estimates as a current product guarantee.
