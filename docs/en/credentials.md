# Credential management

[한국어](../credentials.md) | **English** · [English README](../../README.en.md)

Database passwords are **not saved by default**. In the connection dialog, select encrypted password storage and choose **Save (Ctrl+S)** to persist a password. Connect does not change the storage preference.

## Save and unlock

1. Enter the connection details, database password and alias.
2. Enable encrypted password storage and save.
3. On first use, enter the master password twice. It must contain at least eight characters; prefer a long passphrase.
4. Selecting a saved connection can unlock its stored password. Once unlocked, the vault is reused during the current application process until you explicitly lock it or exit.

Use Ctrl+Enter or the dialog's proceed button to submit the master password, or Esc to cancel. **If you forget the master password, cancel unlocking, enter the database password manually, and connect.** You do not need to reset or save the vault just to connect. Changing the host, port, user, connection mode or service clears an automatically loaded password so it is not sent to another target.

## Manage the vault

Use **Alt+K** from the main or connection screen, or **Tools → Password management**. Tab or Left/Right selects a button; Enter or the mouse activates it.

- **Unlock**: unlock an existing vault or initialize one.
- **Lock**: clear the in-memory key and connection password field without disconnecting the active Oracle session.
- **Change master password**: unlock first, enter a new password twice, and re-encrypt the stored database passwords with a new key and salt.
- **Reset**: after confirmation, delete all stored database passwords. No master password is required; connection profiles and active Oracle sessions remain. Cancel is the safe default.
- **Disable storage for a connection**: clear the encrypted-storage option and save. Other connections remain unchanged; unlocking may be needed to delete the saved entry.

**The master password cannot be recovered.** To store passwords again after forgetting it, reset the vault and re-enter the database passwords. Reset does not erase old backups of the encrypted file. Saving under a new alias creates another profile, following the existing alias behavior.

## Files and shared servers

The default Linux locations are:

| File | Contents |
| --- | --- |
| `~/.config/lazyora/settings.json` | UI preferences |
| `~/.config/lazyora/connections.json` | Aliases, connection details, IDs and password-storage flags; no database passwords |
| `~/.config/lazyora/credentials.enc` | Encrypted database passwords, salt and format metadata |

Other operating systems use their user configuration directory. `LAZYORA_CONFIG` overrides the settings path; related files stay in that directory. Colliding names are disambiguated, including a `secret-` prefix for the vault when necessary. No vault is created when password storage is unused.

Storage is per OS user/configuration directory. People sharing the same OS account or configuration directory also share the vault. Do not store the master password next to it in plaintext.

## Protection and failures

The vault uses Argon2id key derivation and AES-256-GCM authenticated encryption. The master password and a separate password-verification hash are not stored. Linux/macOS files use `0600` and newly created directories use `0700`; Windows follows the configuration directory's ACLs. Locking and exit clear key bytes and input fields, but cannot guarantee erasure of every copy made by Go or the OS, or protection from a server administrator or access to an already-unlocked process.

Invalid master passwords or corrupted vaults are rejected while preserving the existing file. Manual password entry remains available without saving an encrypted password.

Updates use temporary files, synchronization, replacement and conflict checks. Profiles and vaults are separate files, not one filesystem transaction; a crash or disk error can still require recovery. On a conflict, lock and unlock again, or reopen the connection dialog to reload changed profiles. Do not treat a save error as success. Check permissions and available disk space.

After an abnormal exit, a `.lock` file may remain. Stop **all LazyOra instances using that directory** before removing only the stale lock file. Do not delete the profile or encrypted vault as a substitute. See the [detailed implementation and recovery notes (Korean)](../credentials.md).
