# LORA-76 Tuivory v0.10.1 반 줄 그림자 / Half-row shadows

[한국어 적용 기록](../tuivory-sync.md) | [English integration record](../en/tuivory-sync.md)

## 한국어

2026-11-03 사용자 요청으로 공개 `github.com/leechis7/tuivory/go v0.10.1`을 기존 LORA-76 브랜치에 적용했습니다. 태그 `go/v0.10.1`, Origin `287ab1d17e3834c13359645d4a4ec132e83ed345`, Sum `h1:Q/cz8UMFjfANg0Xi/jKtSWcuhtA79vBwW1xs3myr78E=`입니다.

기존 공개 `ShadowView`와 `ShadowAuto`를 그대로 소비해 **오른쪽 1칸·하단 반 줄 높이**를 적용합니다. 반 줄은 Unicode 상단 반 블록 `▀`로 그리지만 터미널 배치에는 1행을 예약합니다. ASCII는 `.`으로 대체합니다. 한글 너비에서 말하는 반각 문자가 아니라 그림자의 시각적 높이입니다. 공통 그리기·색상·문자 겹침 복구는 SDK가 담당하며 앱에 별도 렌더러를 추가하지 않았습니다.

앱의 기존 명시적 1×1 그림자 할당은 SDK의 카드/그림자 배치와 호환됩니다. 반환된 `Card`/`ShadowRegions()`와 `InRect()`를 사용하는 기존 경계를 유지하며, 그림자는 포커스나 클릭을 받지 않습니다. 좁은 화면에서는 할당 가능한 방향만 유지하거나 그림자를 끕니다. 팝업 테두리와 폼·선택 목록·입력 전달을 유지합니다.

변경 소스 비교에서 SDK의 그림자·팝업 기하/견본·검사에 한정된 패치임을 확인했습니다. SQL 분석기·트랜잭션·편집기 동작을 새로 바꾸지 않습니다. [v0.10.0의 전수 재검증](lora-76-tuivory0100.md)은 그 버전의 근거이며 이번 패치에서 다시 전체 전수 조사를 수행했다고 표시하지 않습니다.

### 검증

- `TestPopupBottomShadowUsesSharedHalfRow`: 3테마 × 4색상 프로필 × Unicode/ASCII = 24개 조합. 크기 왕복 후 실제 Show 출력의 `▀`/`.` 및 전경/배경 구별, 클릭 경계를 확인합니다.
- `go test ./...` 통과(UI 21.807초), `go vet ./...` 통과.
- Linux ARM64 `make build` 통과. 바이너리 메타데이터의 SDK 버전과 체크섬 일치 확인.
- 한국어·영어·영문 ASCII 실제 PTY 3모드/크기 변경 33회, 정상 종료 0, 임시 설정 원문 보존. ASCII 팝업에서 비ASCII 문자가 출력되지 않는 검사 포함.
- 관련 동시성 검사와 화면 그림자 셀 확인의 최종 결과는 아래에 기록합니다.

실DB·원격 CentOS/SecureCRT/MobaXterm·물리 폰트·Windows/macOS를 새로 검증한 것은 아닙니다. 개발 바이너리의 앱 버전은 v0.12.1로 유지합니다. 커밋·GitHub 게시·CI·배포는 수행하지 않습니다.

## English

Applied published `github.com/leechis7/tuivory/go v0.10.1` on the existing LORA-76 branch on 2026-11-03. Tag `go/v0.10.1`, Origin `287ab1d17e3834c13359645d4a4ec132e83ed345`, checksum `h1:Q/cz8UMFjfANg0Xi/jKtSWcuhtA79vBwW1xs3myr78E=`.

Existing public ShadowView/ShadowAuto now deliver **one cell right and half a row down**. Unicode uses the upper half block `▀`; terminal layout still reserves a full row. ASCII uses `.`. This refers to visual shadow height, not half-width Korean characters. SDK rendering, coloring and overlap repair own the change; no duplicate consumer renderer was introduced.

The app's explicit 1×1 allocation remains compatible. Preserve returned Card/ShadowRegions geometry and InRect hit testing, noninteractive shadows, narrow-screen clipping/disable behavior, popup borders and input/choice routing. SDK source differences are limited to shadow/popup geometry, examples and tests; no new SQL/transaction/editor change is implied. The [v0.10.0 exhaustive revalidation](lora-76-tuivory0100.md) remains that version's evidence, not a claim that this patch reran the full matrix.

### Verification

New rendered-shadow coverage spans 24 combinations (three themes × four color profiles × Unicode/ASCII), round-trip resizing, physical Show glyphs/contrast and noninteractive shadow boundaries. `go test ./...` passed (UI 21.807s), `go vet ./...` passed, and Linux ARM64 `make build` passed with matching SDK version/checksum in binary metadata. Three actual PTY modes (Korean/English/English ASCII), 33 resizes, normal exit 0 and unchanged settings passed, including an ASCII-only popup assertion. Final targeted race and shadow-cell results are recorded below.

No new live database, remote CentOS/SecureCRT/MobaXterm, physical-font or Windows/macOS certification. The development binary still reports app v0.12.1. No commit, publication, GitHub CI or release performed.

### 최종 확인 / Final confirmation

실제 PTY 접속 팝업에서 한국어·영어 각각 반 블록 91개(오른쪽 위 캡 `▄` 1개, 하단 `▀` 90개)를 확인했고 ASCII에서는 반 블록 0개를 확인했습니다. / Actual PTY connection popups contain 91 half blocks per Unicode language (one upper-right `▄` cap and 90 bottom `▀` cells); ASCII contains none.

관련 팝업·한글 경계·비교표 동시성 배치 실행에서 제품 검사는 통과했으며, 새 검사만 ‘그림자는 두 크기에서만 존재한다’는 잘못된 가정으로 실패했습니다(좁은 크기에도 유효하게 존재해 3회). 이 가정을 수정한 뒤 전체 일반 검사는 위와 같이 통과했고, 수정된 새 반 줄 검사만 race로 다시 실행해 2.833초·종료 0으로 통과했습니다. 배치 race 명령 전체가 수정 후 재실행됐다고 표시하지 않습니다. / The targeted popup/wide-boundary/comparison race batch passed product checks but failed the new test's incorrect assumption that only two sizes retain shadows; the narrow size validly retains one as well. After correcting that assumption, the full ordinary suite passed as above and the revised half-row test passed a separate race rerun in 2.833s with exit 0. The entire race batch was not rerun after this fixture correction.

한영 갱신 문서의 로컬 링크 82개와 `git diff --check`도 통과했습니다. / All 82 local links in updated bilingual documents and `git diff --check` passed.
