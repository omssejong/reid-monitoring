//go:build windows

package util

// patchScriptPath 패치 zip 안에 동봉된 실행 스크립트 경로.
// 윈도우에서는 배치 파일을 사용한다.
const patchScriptPath = `temp\patch_script.bat`
