# lx

[[English](README.en.md)]

> Interface by Human, Logic by AI.

`lx`는 LLM 함수화 방식을 통해 코드의 통제권은 인간이, 구현의 번거로움은 LLM이 담당하는 새로운 프로그래밍 패러다임을 제공합니다.

## Table of Contents
- [Features](#features)
- [How to Use](#how-to-use)
- [Installation](#installation)
- [License](#license)

## Features
- 인터페이스 중심 설계(LLM의 함수화): 개발자는 프로그램의 구조와 데이터 흐름을 설계합니다. `lx`는 개발자가 정의한 함수의 이름, 파라미터, 반환형을 불변의 계약으로 간주합니다. AI는 이 계약 조건을 단 하나라도 어길 수 없으며, 오직 그 내부 로직만을 구현할 수 있습니다.

- 인터페이스 중심 개발의 연속성: 함수의 입출력을 작성하는 순간 해당 로직은 이미 완성된 것으로 간주됩니다. 프로그래머는 세부 구현에 매몰되지 않고 즉시 로직을 작성하며 개발 흐름을 유지할 수 있습니다.

- 개발자의 통제권: 프로그래머는 LLM이 생성한 코드를 직접 확인하고 수정할 수 있으며, 기존 프로그래밍 문법 안에서 LLM을 완전히 통제합니다.

- 토큰 사용 최적화 및 보안 강화: 파일 전체가 아닌 해당 함수의 시그니처와 프롬프트만 LLM에 전달하여 비용을 절감하고 보안을 강화합니다.

- 함수 단위 격리 및 안전성: LLM은 오직 정의된 `lx` 함수에서만 동작합니다. 프로젝트의 전역 구조를 오염시키지 않습니다.

- 계층적 설정: `lx`는 현재 디렉토리의 설정을 최우선으로 하며, 없을 경우 홈 디렉토리의 전역 설정을 따릅니다. 이를 통해 프로젝트의 성격에 따라 각기 다른 LLM 모델이나 API 키를 사용하는 '프로젝트별 최적화'가 가능합니다.

- 투명한 의존성 리포트: AI가 외부 라이브러리를 사용할 경우 `// lx-dep` 주석으로 코드로써 보고합니다. 도구가 임의로 패키지를 설치하지 않으며, 개발자가 직접 코드를 보고 적용 및 설치 여부를 판단합니다.

## How to Use

### Definition
- lx Function: lx를 통해 로직이 구현되는 함수를 lx 함수라고 약속합니다.
- lx Marker: lx 함수 내부에서 LLM에게 구현할 로직의 내용을 전달하는 구현 명세를 lx 마커라고 약속합니다.
- lx Tool: lx 함수에 명시된 추상적 계약을 실제 로직으로 치환하여 실체화하는 도구를 lx 도구라고 약속합니다.
- lx Configuration: lx 도구가 로직을 생성할 때 사용할 LLM Provider, Model, LLM API KEY를 정의하는 환경 명세를 lx 설정이라고 약속합니다.
- lx Dependency: lx 도구가 로직을 생성할 때 필요한 시스템 도구을 lx 의존성이라고 약속합니다.

### lx Dependency
lx는 생성된 코드의 문법적 정합성을 확보하고 품질을 높이기 위해 외부 포맷팅 도구를 내부적으로 호출합니다.
lx는 패키지 매니저가 아니며 시스템 환경을 오염시키지 않는다는 단일 책임 원칙을 고수하므로, lx를 설치할 때 아래의 도구들은 같이 설치되지 않습니다.
따라서 아래의 도구들은 사용자가 수동으로 설치해야 합니다.

- Go: goimports
- Python: ruff
- JavaScript: prettier

### lx Configuration
lx를 사용하기 앞서 LLM의 API Key와 사용할 모델명을 명시해야 합니다.
lx-config.yaml 파일을 만들어서 다음 두 경로 중 원하는 곳에 위치하면 됩니다.
만약 두 경로 모두 파일을 위치한다면, 로컬 설정 파일이 우선적으로 적용됩니다.

- 전역 경로: 사용자의 홈 디렉토리(`~/lx-config.yaml`)에 파일을 위치하면, 모든 프로젝트에서 공통으로 같은 LLM를 사용할 수 있습니다.
- 로컬 경로: 프로젝트의 루트 디렉토리(`./lx-config.yaml`)에 파일을 위치하면, 특정 프로젝트만 다른 모델을 쓰거나 다른 API 키를 써야 할 때 사용할 수 있습니다.

`lx-config.yaml` 파일은 반드시 해당 형식으로 작성되어야 합니다.
```yaml
# lx-config.yaml
provider: "gemini"
api_key: "foo"
model: "bar"
```

`lx`가 현재 지원하는 LLM provider는 다음과 같습니다.

- gemini

### lx Marker
lx 마커는 두 가지의 스타일이 제공됩니다.

#### Comment lx Marker
별도의 라이브러리 의존성 없이 사용하고 싶을 때 적합합니다.
반드시 주석 안에 lx("프롬프트 내용") 코드가 포함되어야 합니다.
단순히 일반적인 주석을 쓰는 것만으로는 인식되지 않습니다.

```go
// lx("한국식 날짜로 변환") (O)

// 이 함수를 한국식으로 바꿔줘 (X)
```

#### Function lx Marker
에디터의 자동완성 및 정적 분석 기능을 활용하고 싶을 때 사용합니다.
제공되는 언어별 라이브러리(예: lxgo)에 맞게 작성해야 합니다.

```go
lx.Generate("한국식 날짜로 변환") // Go
```

함수 마커를 지원하는 프로그래밍 언어는 다음과 같습니다.

- Golang: https://github.com/chebread/lxgo

### lx Function
lx 함수는 다음의 엄격한 규칙을 따릅니다

- 네이밍 컨벤션: lx에 의해 생성됨을 명시하기 위해 함수 이름은 LX_ 또는 lx_로 시작하는 것을 권장합니다.

- 명시적 인터페이스: 입출력 데이터는 lx.Generate의 인수가 아닙니다. 함수의 인자 타입, 반환 타입으로 명확히 정의되어야 합니다.

- 파괴적 처리: lx 함수 내부에 작성된 사용자 정의 로직은 반영되지 않으며 완전히 무시됩니다. 오직 lx.Generate()의 프롬프트 내용만 LLM에게 전달됩니다.

- 문법적 요구사항: lx 함수 내부에는 컴파일러와 IDE의 오류를 방지하기 위해 최소한의 프로그래밍 언어 문법적 요소만 작성합니다.

```go
package test

import (
	"fmt"

	"github.com/chebread/lxgo"
)

func main() {
	var year string = "2025-01-02"
	foo := LX_GetYear(year)

	var age = 30
	bar := LX_GetAge(age)

	fmt.Println(foo, bar)
}

func LX_GetYear(year string) (result string) {
	lx.Generate("yyyy-dd-mm 형식을 한국식 날짜로 변환")
	return
}

func LX_GetAge(year int) (result string) {
	// lx("한국식 나이를 만 나이로 변환")
	return
}
```

### lx Function Control
lx 함수 내부에는 개발자 로직의 모두 무시되어 작성할 수 없으므로, 로직 실행 전후에 데이터를 통제하거나 추가적인 처리가 필요하다면 반드시 다른 함수로 lx 함수를 래핑해야 합니다. 이것이 lx가 추구하는 "개발자의 통제권 상에서의 개발" 방식입니다.

```go
package test

import (
	"fmt"

	"github.com/chebread/lxgo"
)

func main() {
	var year string = "2025-01-02"
	res := ParseYear(year)

	fmt.Println(res)
}

func ParseYear(year string) string {
	koreanYear := LX_GetYear(year)
	foo := fmt.Sprintf("오늘은 %v 입니다!", koreanYear)
	return foo
}

func LX_GetYear(year string) (result string) {
	lx.Generate("yyyy-dd-mm 형식을 한국식 날짜로 변환")
	return
}
```

### lx Tool
작성된 lx 함수와 마커는 lx 도구를 실행하기 전까지는 아무런 기능도 하지 않는 빈 껍데기입니다.
lx 명령어를 실행해야 비로소 추상적인 설계가 구체적인 코드로 변환되어 함수 본문에 주입됩니다.

Mac 사용자의 경우 lx 도구는 아래의 Homebrew 명령어를 통해 간편하게 설치할 수 있습니다:
```bash
brew tap chebread/lx
brew install lx
```

타 운영체제는 현재 지원하지 않습니다.

### 코드 주입 및 실체화
lx 도구를 실행하면 프로젝트 내의 lx 함수 및 lx 마커를 탐색하여 LLM에 전송하고, 반환된 실제 로직으로 기존의 마커 코드를 덮어씌웁니다.
주의할 점은, lx 실행 시 lx 함수가 컴파일 가능한 실제 코드로 바뀌게 되므로 이 과정은 실제 소스 코드를 변경합니다.

### 스마트 생성
lx는 이미 코드로 변환된 함수를 기억합니다.
lx 도구로 인해 실제 코드가 생성된 lx 함수에 대해 lx를 반복 실행하더라도, lx 도구는 절대로 LLM을 다시 호출하지 않습니다.
언제든 걱정 없이 안심하고 lx 명령어를 실행하세요.

### 컴파일 전 필수 실행
lx 도구는 오직 LLM 코드 생성이라는 단일 책임만 수행합니다.
프로그램의 빌드와 실행은 전적으로 개발자의 몫입니다.
컴파일 또는 Run 하기 직전에 lx가 먼저 실행되도록 파이프라인을 구성하는 것을 권장합니다.

### 4. 안전한 의존성 관리
lx는 생성된 로직이 외부 라이브러리를 사용하더라도, 외부 패키지를 임의로 설치하지 않습니다.
대신 다음 두 가지 방법으로 개발자에게 보고합니다.

- Code: 생성된 코드 상단에 `// lx-dep: ...` 주석 명시

- Output: 터미널 표준 출력으로 설치 필요 목록 리포트

개발자는 이 리포트를 보고 직접 라이브러리를 설치(go get 등)하여 의존성을 통제하면 됩니다.
이는 개발자가 프로젝트의 모든 의존성을 완벽히 통제해야 한다는 lx의 설계 철학에 기인합니다.

## License
**AGPL-3.0 LICENSE** &copy; 2026 Cha Haneum

본 프로젝트는 **AGPL-3.0** 라이선스를 따릅니다.
사용 및 배포 시 라이선스 조항을 반드시 확인하시기 바랍니다.
