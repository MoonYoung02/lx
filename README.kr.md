# lx
[[English](README.md)]

`lx`는 LLM을 프로그래밍 언어의 함수처럼 다룰 수 있게 해주는 LLM Functionization 도구입니다.
`lx`의 핵심은 LLM을 사용하되, LLM 그 자체와 그 결과물을 완벽하게 프로그래머의 통제권 아래 두는 것에 있습니다.

## Table of Contents
- [Features](#features)
- [Configuration](#configuration)
- [How to Use](#how-to-use)
- [Installation](#installation)
- [Supported Languages](#supported-languages)
- [Supported LLMs](#supported-LLMs)
- [License](#license)

## Features
* **LLM의 함수화**: 자연어 프롬프트를 실제 실행 가능한 코드로 변환하여 함수 본문에 주입합니다.
* **프로그래머의 통제권**: 프로그래머는 LLM이 생성한 코드를 직접 확인하고 수정할 수 있으며, 기존 프로그래밍 문법 안에서 LLM을 통제합니다.
* **개발의 연속성**: 함수의 입출력을 정의하는 순간, 그 함수의 구현은 이미 끝난 것이나 다름없습니다. 로직 구현은 나중에 lx 명령 한 번으로 일괄 주입하면 됩니다. 프로그래머는 즉시 다음 비즈니스 로직을 이어서 작성할 수 있습니다.
* **함수 단위 격리**: LLM은 오직 당신이 허락한 함수 내부 공간에서만 동작합니다. 프로젝트의 전역 구조나 다른 소스 코드를 오염시킬 걱정이 없습니다.
* **컴파일 타임에 생성**: 런타임에 LLM을 호출하는 것이 아니라, 개발 단계에서 코드를 생성하므로 실행 속도가 빠르고 안정적입니다.

## Configuration
`lx`를 사용하려면 프로젝트 루트에 `lx-config.yaml` 파일이 필요합니다. 현재는 Google Gemini 모델만 지원합니다.

```yaml
provider: "gemini"
api_key: "YOUR_GEMINI_API_KEY"
model: "gemini-..."

```

## How to Use

### 1. Library 설치
Go 언어에서 `lx.Generate` 함수를 사용하기 위해 의존성을 추가해야 합니다. 현재는 Go 언어만 지원합니다.

```bash
go get github.com/chebread/lxgo

```

### 2. 함수 작성
lx는 함수 단위의 코드 주입(Function Body Injection) 도구입니다.

### I. "No Function, No Action" 원칙
lx는 Go의 AST를 분석하여 func 선언 내부에 lx.Generate 깃발이 있는 경우에만 동작합니다. 함수 밖의 프롬프트는 안전하게 무시됩니다.

### II. 파괴적 주입에 대한 주의사항
lx를 실행하는 순간, 해당 함수 내부에 있던 기존의 모든 코드는 삭제되고 AI가 생성한 로직으로 대체됩니다.

실행 전: 당신이 임시로 적어둔 코드는 lx 실행 시 사라집니다.

실행 후: AI가 생성한 코드가 주입된 시점부터 개발자의 직접적인 수정 및 접근이 가능해집니다.

### III. 통제권 확보 방법
만약 lx가 생성할 로직 전후로 개발자의 커스텀 로직을 유지하고 싶다면, Wrapping 방식을 사용하세요.

### IV. 함수 작성 방법
네이밍 컨벤션 (권장): 함수의 이름을 lx_ 또는 LX_로 시작하여 AI 관리 대상임을 명시하세요.

명시적 리턴: 만약 반환값이 있는 함수라면 lx 실행 전 컴파일 에러를 방지하기 위해 빈 `return` 문 같은 것을 작성해 두어야 합니다. 이는 전적으로 프로그래밍 언어 문법에 기초합니다.

```go
package test

import "github.com/chebread/lxgo"

// 함수의 이름은 LX_ 혹은 lx_로 시작하는 것을 권장합니다.
func LX_Year(year string) (result string) {
    // lx.Generate는 코드를 생성하기 위한 깃발 역할을 합니다.
    lx.Generate("yyyy-dd-mm 형식을 한국식 날짜로 변환")
    
    // Go 문법상 반환값이 있다면 반드시 return 문이 존재해야 합니다.
    return
}

```

### 3. lx 도구 실행

`go run .`을 실행하기 전에 반드시 `lx` 명령어를 실행해야 합니다. 만약 실행하지 않으면 해당 함수들은 아무런 동작도 하지 않는 빈 상태로 남게 됩니다.

```bash
# 현재 디렉토리의 모든 .go 파일을 분석하여 AI 코드로 교체합니다.
> lx .

```

`lx`를 실행하면 함수 내부의 `lx.Generate` 호출부를 포함한 모든 코드가 **AI가 생성한 실제 로직으로 덮어씌워집니다.** 그 이후부터는 프로그래머가 생성된 코드를 직접 수정하거나 통제할 수 있습니다.

> **Note**: 만약 AI의 생성을 통제하고 싶다면, `lx` 함수를 직접 수정하는 대신 해당 함수를 Wrapping 하는 방식으로 설계하세요.

## Installation

### On macOS

Homebrew를 통해 간편하게 설치할 수 있습니다:

```bash
brew tap chebread/lx
brew install lx
```

### For other OS

1. [GitHub Releases](https://github.com/chebread/lx/releases) 페이지를 방문합니다.
2. 운영체제와 아키텍처에 맞는 파일을 다운로드합니다.
3. 압축을 해제하고 `lx` 실행 파일을 시스템 PATH 환경 변수에 추가합니다.

## Supported Languages

* **Golang**
* *More languages coming soon...*

## Supported LLMs

* **Gemini**
* * *More LLMs coming soon...*

## License

**AGPL-3.0 LICENSE** &copy; 2026 Cha Haneum

본 프로젝트는 **AGPL-3.0** 라이선스를 따릅니다. 사용 및 배포 시 라이선스 조항을 반드시 확인하시기 바랍니다.
