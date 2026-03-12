# Si-Gnal (시-그날)

![Si-Gnal](assets/img/si-gnal.png)

시사랑 사이트에서 무작위 시를 가져와서 음성으로 읽어주는 프로젝트입니다.
백그라운드에서 지속적으로 무작위 시를 가져와 음성 파일(`.wav`)로 생성하고 대기열을 유지하며, `/api/play` API 호출 시 서버가 실행 중인 기기의 스피커에서 즉시 해당 시를 낭독해 줍니다. 

## 주요 기능

- **무작위 시 추출**: 시사랑 사이트에서 무작위로 시를 가져옵니다.
- **AI 대본 정리**: Gemini API (Gemma 모델)를 활용하여 시의 내용 중 불필요한 부분(제목, 작가 등 부가정보)을 정제하고 낭독용 스크립트로 자연스럽게 변환합니다.
- **음성 합성(TTS)**: `supertonic`, `htgo` 등의 엔진을 통해 텍스트를 고품질 오디오 파일(`.wav`)로 변환합니다.
- **백그라운드 생성(Batch)**: API 호출 시 지연 없이 바로 재생될 수 있도록 백그라운드에서 여러 개의 오디오 파일을 미리 생성하여 대기열을 유지합니다.
- **음향 효과 적용**: 생성된 음성에 아날로그 전화선 효과(wire-phone) 및 노이즈 효과를 추가할 수 있습니다.

## 환경 설정

Si-gnal 서버 구동을 위해 다음 환경변수들이 설정되어야 합니다:

1. **Gemini API Key 설정** (필수)
   시 대본 정리를 위한 AI 동작에 Gemini API 키가 필요합니다.
   ```bash
   export GEMINI_API_KEY="your-gemini-api-key"
   ```

2. **Supertonic 모델 다운로드**
   `supertonic` 엔진을 사용하려면 Hugging Face의 [supertonic-2](https://huggingface.co/Supertone/supertonic-2) 모델 파일들이 필요합니다. Git LFS를 이용하여 다음 스크립트를 실행해 `assets/supertonic2` 디렉토리에 다운로드합니다.
   ```bash
   ./scripts/download_supertonic.sh
   ```

3. **Supertonic TTS (Mac 환경) 설정**
   Mac 환경에서 OnnxRuntime을 의존하는 기본 `supertonic` TTS 엔진을 정상적으로 사용하려면 다음 라이브러리 경로 환경변수가 필요합니다.
   ```bash
   export ONNXRUNTIME_LIB_PATH=$(brew --prefix onnxruntime 2>/dev/null)/lib/libonnxruntime.dylib
   ```

> `.zshrc` 또는 터미널 프로파일에 위 항목들을 추가해두면 편리합니다.

## 사용법

### 1. 웹 서버 실행

프로젝트 루트 디렉토리에서 다음 명령어로 서버를 실행합니다.

```bash
go run cmd/server/main.go [옵션]
```

**실행 옵션 (Flags)**

| 플래그 | 타입 | 기본값 | 설명 |
|--------|--------|--------|------|
| `-b` | `int` | `5` | 미리 생성하여 대기열에 유지할 wav 파일의 개수 (배치 사이즈) |
| `-e` | `bool` | `false` | 출력 결과에 오래된 무전기/전화선(wire-phone) 음향 효과 적용 |
| `-n` | `bool` | `false` | 오디오에 백그라운드 노이즈 추가 (`-e` 효과와 시너지) |
| `-p` | `bool` | `false` | (Legacy) 생성된 wav 파일을 서버 내부 큐에 넣는 대신 보류하고 직접 스피커로 재생 |
| `-t` | `string` | `"supertonic"` | 사용할 TTS 엔진 지정 (`supertonic`, `htgo` 중 선택) |
| `-m` | `bool` | `false` | 생성된 wav 파일을 디스크 대신 메모리에 저장 |

**실행 예시:**
> 노이즈와 전화기 효과를 넣고, 미리 3개의 음성을 확보한 상태로 서버 시작
```bash
go run cmd/server/main.go -e -n -b 3
```

### 2. REST API

서버가 실행되면(기본 포트: 8080), 아래와 같은 API들을 통해 대기열을 관리하고 시를 감상할 수 있습니다.

#### 큐 상태 확인
- **`GET /api/poem`**: 현재 대기열에 있는 모든 시와 생성된 파일 정보를 JSON 리스트로 반환합니다.

#### 시 정보 조회 (Peek)
- **`GET /api/poem/head`**: 대기열의 가장 첫 번째 시 정보를 반환합니다. (큐에서 제거되지 않음)

#### 시 꺼내기 (Pop)
- **`GET /api/poem/pop`**: 대기열의 가장 첫 번째 시 정보를 반환하고 큐에서 제거합니다.

---

#### 추가 기능 (Query Parameters)
`head` 또는 `pop` 엔드포인트 호출 시 `play` 쿼리 파라미터를 통해 추가 동작을 지정할 수 있습니다.

- **스피커로 재생**: `?play=speaker`
  - 명령: `curl http://localhost:8080/api/poem/pop?play=speaker`
  - 정보(JSON)를 반환함과 동시에 서버 기기의 스피커로 시를 낭독합니다.
- **오디오 파일 스트림**: `?play=wav`
  - 명령: `curl http://localhost:8080/api/poem/pop?play=wav --output poem.wav`
  - JSON 대신 실제 생성된 `.wav` 오디오 파일 데이터를 직접 반환합니다.

#### 재생 제어
- **`POST /api/stop`**: 현재 스피커로 재생 중인 낭독을 중단하고 파일을 정리합니다.

---

### 기타 유틸리티

단순히 지정한 형식에 맞춰 무작위 시를 콘솔 환경에서 확인하고 싶을 때 유용한 도구도 포함되어 있습니다.

```bash
go run test_cmd/random_poem/main.go -f <출력형식>
```
위 명령어를 사용하면 무작위 시 한 편과 다듬어진 AI 낭송 대본을 콘솔로 출력합니다. (출력형식 지원: `txt`, `yaml`, `json`)
