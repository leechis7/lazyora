# LORA-76 Tuivory v0.10.2 Windows 수정 / Windows fix

[한국어 적용 기록](../tuivory-sync.md) | [English integration record](../en/tuivory-sync.md)

## 한국어

2026-11-03, 승인된 v0.12.2 배포를 기존 PR #70에서 재개합니다. 공개 `github.com/leechis7/tuivory/go v0.10.2`, 태그 `go/v0.10.2`, Origin `5e95114869356297c38cf69dc3b1cc46675475eb`, Sum `h1:iPVsFzd0TLAsADWvMRnP6S0kD0OsZsZzglGAyOeob+Y=`를 확인했습니다.

v0.10.1의 공통 생성 코드가 Windows에 없는 `native.NewStdIoTty`를 무조건 참조하여 [이전 CI](https://github.com/leechis7/lazyora/actions/runs/41554682807)의 Windows 두 작업 및 전체 대상 빌드가 실패했습니다. Linux race·vet·AMD64/ARM64 빌드와 macOS UI 검사는 통과했지만 배포 성공으로 계산하지 않았습니다. SDK가 v0.10.1을 회수했으며 현재 의존성으로 사용하지 않습니다.

v0.10.2는 루트/terminal 생성 단계에 플랫폼별 build constraints를 보존합니다. Unix 동작과 공개 이름을 유지하고 Windows에서는 backend에 없는 함수를 노출하지 않습니다. LazyOra에 SDK 복사·로컬 shim·플랫폼 예외를 추가하지 않았으며 모든 배포 OS를 유지합니다. 비교 소스에서 공통 렌더러/그림자/SQL 분석기는 변경되지 않았음을 확인했습니다. 이전 그림자와 한글·대비 회귀는 기존 검사를 그대로 적용합니다.

로컬 기본 빌드는 Linux ARM64만 수행하고, Windows·macOS 및 전체 아키텍처는 GitHub CI/Release에서 확인합니다. 로컬 결과와 PR/배포 상태는 Jira LORA-76에 기록합니다. 실DB·원격 터미널·물리 폰트 검증으로 확대해 주장하지 않으며 LORA-76 전체 배치/편집기 전환의 완료와도 구분합니다.

## English

Resume the authorized v0.12.2 deployment on existing PR #70 on 2026-11-03. Verified public Go v0.10.2, tag `go/v0.10.2`, Origin `5e95114869356297c38cf69dc3b1cc46675475eb`, checksum `h1:iPVsFzd0TLAsADWvMRnP6S0kD0OsZsZzglGAyOeob+Y=`.

The [previous CI](https://github.com/leechis7/lazyora/actions/runs/41554682807) failed both Windows jobs/all-target builds because v0.10.1 unconditionally forwarded the unavailable native.NewStdIoTty. Linux race/vet/AMD64/ARM64 builds and macOS UI passed, but this was not a successful release. Upstream retracted v0.10.1; it is no longer consumed.

The new SDK preserves platform constraints in root/terminal generation, maintaining existing Unix behavior and omitting unsupported backend APIs on Windows. No consumer SDK copy/shim/platform bypass or removed OS support. Source comparison shows no change to common rendering/shadows/SQL analysis; retain existing regression checks.

Local default build remains Linux ARM64; GitHub CI/Release validates Windows/macOS and all supported architectures. Record final local/CI/publication results in LORA-76. This is not live-DB/remote-terminal/physical-font certification or completion of the broader layout/editor migration.

로컬 `go test ./...` 통과(UI 17.874초), `go vet ./...` 및 Linux ARM64 빌드 통과. 로컬 문서 링크 450개·diff 검사도 통과했습니다. Windows 및 전체 대상 확인은 PR CI에서 수행합니다. / Local full ordinary tests passed (UI 17.874s), as did vet, Linux ARM64 build, 450 documentation links and diff checks. Windows/all-target verification is delegated to PR CI.
