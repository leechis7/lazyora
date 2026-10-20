# LazyOra의 tview 렌더링 확장

출처: [rivo/tview v0.42.0](https://github.com/rivo/tview/tree/v0.42.0).
원본 모듈의 루트 Go 소스·go.mod·go.sum·LICENSE.txt를 보존했습니다. MIT 라이선스는 LICENSE.txt에 있습니다.

변경은 textarea.go에 한정합니다. `SetTextStyleFunc`는 원문 UTF-8 바이트 위치별로 **선택되지 않은** 글자 스타일을 지정합니다. 첫 화면 위치를 piece chain에서 한 번 찾고 보이는 grapheme별로 바이트 위치를 증가시킵니다. 기존 편집·커서·줄바꿈·스크롤·선택 렌더링을 유지하며 선택 스타일이 우선합니다. 입력이나 SQL 원문에 색상 태그를 삽입하지 않습니다.

프로젝트 go.mod의 replace로 이 디렉터리를 사용합니다. 원본 버전을 올릴 때 textarea.go의 작은 차이를 다시 검토하고, 루트 테스트의 구문 색상·입력·선택·한글·스크롤 검사와 전체 회귀를 실행해야 합니다. 공개 모듈 캐시를 직접 수정하지 않습니다.
