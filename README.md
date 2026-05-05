# Si-Gnal (시-그날)

![Si-Gnal](assets/img/si-gnal.png)

시사랑 사이트에서 무작위 시를 가져와서 음성으로 읽어주는 프로젝트입니다.  
백그라운드에서 지속적으로 무작위 시를 가져와 음성 파일(`.wav`)로 생성하고 대기열을 유지하며, `GET /api/poem/head` 또는 `GET /api/poem/pop`에 `?play=speaker`를 붙이면 서버가 실행 중인 기기의 스피커에서 해당 시를 낭독합니다.

## 주요 기능

- **무작위 시 추출**: 시사랑 사이트에서 무작위로 시를 가져옵니다.
- **AI 대본 정리**: OpenAI 호환 Chat API를 사용해 본문 정제와 낭독용 스크립트(`ReadingScript`)를 생성합니다.
- **음성 합성(TTS)**: [Supertonic](https://huggingface.co/Supertone/supertonic-2) ONNX 기반 엔진으로 텍스트를 `.wav`로 변환합니다.
- **백그라운드 생성**: API 호출 시 바로 재생할 수 있도록 미리 여러 개의 오디오를 만들어 큐에 넣습니다.
- **음향 효과**: 설정에서 wire-phone(아날로그 전화선) 효과와 노이즈를 켤 수 있습니다.

## 요구 사항

- Go **1.25.5** 이상 (`go.mod` 기준)
- **OpenAI 호환 API 키** (기본은 OpenAI 공식 엔드포인트; `base_url`로 호환 서비스 지정 가능)
- Supertonic 사용 시: Hugging Face의 [supertonic-2](https://huggingface.co/Supertone/supertonic-2) 에셋 및 (macOS 등에서) ONNX Runtime 동적 라이브러리

## 환경 설정

### 1. 설정 파일

`cmd/server/config_sample.yaml`을 복사해 프로젝트 루트(또는 실행 시 작업 디렉터리)에 `config.yaml`로 둡니다.

```bash
cp cmd/server/config_sample.yaml config.yaml
```

값에 `${환경변수명}`을 쓰면 로드 후 환경에서 치환됩니다.

### 2. OpenAI API

`config.yaml`의 `openai` 섹션에 키를 넣거나, 샘플처럼 `${OPENAI_API_KEY}`만 두고 셸에서 설정합니다.

```bash
export OPENAI_API_KEY="sk-..."
```

선택: `openai.base_url`을 비우면 라이브러리 기본값(`https://api.openai.com/v1`)을 쓰고, 다른 호환 제공자를 쓰려면 URL을 지정합니다. `openai.model` 기본값은 `gpt-4o-mini`입니다.

### 3. Supertonic 모델

```bash
./scripts/download_supertonic.sh
```

모델은 기본적으로 `assets/supertonic2` 아래에 두며, `config.yaml`의 `tts.supertonic.onnx_dir`, `tts.supertonic.voice_style`으로 경로를 바꿀 수 있습니다.

### 4. macOS에서 ONNX Runtime

```bash
export ONNXRUNTIME_LIB_PATH=$(brew --prefix onnxruntime 2>/dev/null)/lib/libonnxruntime.dylib
```

> `.zshrc` 등에 넣어두면 편합니다.

## 설정 항목 (`config.yaml`)

| 키 | 설명 |
|----|------|
| `server.listen` | HTTP 바인딩 주소 (기본 `:8080`) |
| `poem_queue.batch` | 미리 만들어 둘 wav 개수 (기본 `5`) |
| `poem_queue.use_memory` | `true`면 wav를 디스크 대신 메모리에 보관 |
| `openai.base_url`, `openai.api_key`, `openai.model` | OpenAI 호환 클라이언트 설정 |
| `tts.engine` | 현재 `supertonic`만 지원 |
| `tts.supertonic.onnx_dir` | ONNX 디렉터리 |
| `tts.supertonic.voice_style` | 음성 스타일 JSON 경로 |
| `wirephone.enabled` | 전화선 효과 적용 여부 |
| `wirephone.add_noise` | 효과에 노이즈 추가 (`enabled`와 함께 사용) |

## 사용법

### 웹 서버 실행

프로젝트 루트에서 `config.yaml`이 보이도록 실행합니다.

```bash
go run ./cmd/server -config config.yaml
```

`-config`를 생략하면 기본값 `config.yaml`입니다.

### REST API

서버 기본 주소는 설정의 `server.listen`입니다 (예: `http://localhost:8080`).

| 메서드·경로 | 설명 |
|-------------|------|
| `GET /api/poem` | 대기열 전체(JSON) |
| `GET /api/poem/head` | 맨 앞 항목 조회 (큐에서 제거하지 않음) |
| `GET /api/poem/pop` | 맨 앞 항목을 꺼냄 |
| `POST /api/stop` | 스피커 재생 중단 |

`head` / `pop`에 쿼리 `play`를 붙일 수 있습니다.

- **`?play=speaker`**: JSON 응답과 함께 서버 머신 스피커로 재생. 이미 재생 중이면 `409 Conflict`일 수 있습니다.
- **`?play=wav`**: 응답 본문이 `audio/wav` 바이너리. `pop`인 경우 디스크 파일 모드에서는 스트림 후 파일을 정리합니다.

예:

```bash
curl "http://localhost:8080/api/poem/pop?play=speaker"
curl "http://localhost:8080/api/poem/pop?play=wav" --output poem.wav
```

### 기타 CLI

**무작위 시만 콘솔/파일로 확인** (`test_cmd/random_poem`):

```bash
go run ./test_cmd/random_poem -f txt
go run ./test_cmd/random_poem -f yaml -o poem.yaml
```

선택적으로 AI 교정·낭독 대본을 쓰려면 다음을 설정합니다. `SIGNAL_OPENAI_BASE_URL`이 비어 있으면 AI 단계는 건너뜁니다.

- `SIGNAL_OPENAI_BASE_URL`
- `SIGNAL_OPENAI_API_KEY`
- `SIGNAL_OPENAI_MODEL` (비우면 `gpt-4o-mini`)

**TTS 단독 스모크 테스트** (`test_cmd/test_tts`): Supertonic으로 짧은 문장 또는 `-i` 파일을 wav로보냅니다. ONNX 경로·라이브러리는 서버와 동일하게 맞춥니다.

### Systemd (라즈베리 파이 등)

`scripts/systemd` 예제와 [scripts/systemd/README.md](scripts/systemd/README.md)를 참고하세요.
