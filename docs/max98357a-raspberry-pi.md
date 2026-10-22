# Adafruit MAX98357A I2S Class-D 앰프 사용법 (Raspberry Pi)

> 출처: [Adafruit Learn — Adafruit MAX98357 I2S Class-D Mono Amp](https://learn.adafruit.com/adafruit-max98357-i2s-class-d-mono-amp/raspberry-pi-usage)
> (개요·핀아웃·Raspberry Pi 배선·설치·테스트 페이지 내용을 정리)

Raspberry Pi에서 I2S 디지털 오디오를 스피커로 출력하기 위한
MAX98357A 기반 3W Class-D 모노 앰프 브레이크아웃(Adafruit 3006)의 배선,
설정, 테스트 방법을 정리한 문서이다. Si-Gnal에서는 이 앰프를 Pi의
기본 ALSA 출력 장치로 설정해 `player`(또는 `?play=speaker`) 재생에 사용한다.

## 1. 개요 및 사양

MAX98357A는 I2S DAC와 Class-D 앰프가 한 칩에 들어 있어, I2S 디지털
입력을 받아 스피커를 직접 구동한다. 아날로그 입력은 없으며, **I2S와
I2C는 전혀 다른 프로토콜**이므로 혼동하지 않는다.

| 항목 | 값 |
|------|----|
| 출력 | 3.2W @ 4Ω (5V, 10% THD) / 1.8W @ 8Ω (5V, 10% THD) |
| 전원 | 2.7–5.5V DC (로직·앰프 공용, 5V 권장) |
| 전원 전류 | 최대 약 650mA → **800mA 이상** 공급 가능한 전원 권장 |
| 샘플레이트 | 8kHz–96kHz (I2S) |
| MCLK | 불필요 (연결하지 않음) |
| 게인 | 핀 선택 3 / 6 / 9 / 12 / 15 dB (기본 9dB) |
| PSRR | 77dB typ @ 1kHz |
| 기타 | 클릭·팝 억제, 과열·과전류 보호 |

스테레오 버전(Dual MAX98357A)도 기능은 동일하고 출력 채널 수만 다르다.

### 주의사항

- **스피커 임피던스는 4Ω 이상**이어야 한다. 동축(moving coil) 스피커 전용이다.
- 출력은 **Bridge-Tied** 방식이다. 스피커 `+`/`-` 모두 GND가 아니므로
  **출력 단자를 GND나 다른 앰프 입력에 연결하면 안 된다.**
  프리앰프로 사용할 수 없다. 출력은 약 300–330kHz PWM 구형파이며,
  스피커 코일의 인덕턴스가 저역통과 필터 역할을 해 가청 신호로 평균화된다.
- 헤드폰 잭 품질보다 깨끗한 출력을 얻을 수 있고, 오디오 출력이 없는
  Pi Zero에서도 사용할 수 있다.

## 2. 핀 설명

| 핀 | 기능 |
|----|------|
| `Vin` | 전원 2.5–5.5V. 3.3V 로직 기기에서도 앰프 전원은 5V로 공급하는 것이 출력 확보에 유리하다 |
| `GND` | 접지 |
| `LRC` (LRCLK) | Left/Right Clock — 좌/우 채널 선택 클럭 |
| `BCLK` | Bit Clock — 데이터 판독 클럭 |
| `DIN` | Data In — 좌/우 데이터가 이 핀 하나로 전송됨 |
| `GAIN` | 게인 설정 (아래 표) |
| `SD` | 셧다운 / 출력 채널 모드 설정 (아래 표) |
| 스피커 단자 | 3.5mm 스크루 터미널 블록. 스피커 `+`/`-`를 그대로 연결 |

I2S 신호는 3.3V/5V 로직 모두 사용 가능하다.

### GAIN — 게인 설정

| 연결 | 게인 |
|------|------|
| 100kΩ 저항으로 `GAIN`–`GND` | 15dB |
| `GAIN`을 `GND`에 직접 연결 | 12dB |
| **미연결 (기본값)** | **9dB** |
| `GAIN`을 `Vin`에 직접 연결 | 6dB |
| 100kΩ 저항으로 `GAIN`–`Vin` | 3dB |

게인 변경 후에는 전원을 껐다 켜야 적용될 수 있다.

### SD — 셧다운 / 채널 모드

`SD` 핀 전압으로 셧다운과 출력 채널이 결정된다.

| `SD` 전압 | 동작 |
|-----------|------|
| < 0.16V (`GND` 직접 연결) | 셧다운 |
| 0.16–0.77V | (Left+Right)/2 — 스테레오 평균 (모노 출력 기본값) |
| 0.77–1.4V | Right 채널만 |
| > 1.4V | Left 채널만 |

브레이크아웃에는 `SD`–`Vin` 사이에 1MΩ 저항이 있어, 5V 전원 사용 시
기본으로 스테레오 평균 출력이 된다. 내부에 100kΩ 풀다운이 있으므로
좌/우 채널만 쓰려면 풀업 저항 값을 전원 전압에 맞춰 조정해야 한다.

## 3. Raspberry Pi 배선

필요한 신호는 I2S 3선 + 전원 2선, 총 5선이다.

| 앰프 핀 | Raspberry Pi | 물리 핀 | 비고 |
|---------|--------------|---------|------|
| `Vin` | 5V | Pin 2 또는 4 | Pi 전원 예산에 **500mA 이상 추가** 확보 |
| `GND` | GND | Pin 6 등 | |
| `DIN` | GPIO 21 (PCM DOUT) | Pin 40 | |
| `BCLK` | GPIO 18 (PCM CLK) | Pin 12 | |
| `LRCLK` (`LRC`) | GPIO 19 (PCM FS) | Pin 35 | |
| `GAIN` | 미연결 | — | 기본 9dB. 바꾸려면 §2 표 참고 |
| `SD` | 미연결 | — | 보드 내장 저항으로 스테레오 평균이 기본값 |

스피커는 터미널 블록에 극성대로 연결한다. 헤더·터미널 블록을 쓰려면
납땜이 필요하다.

## 4. Raspberry Pi OS 설정

### 4.1 자동 설치 스크립트 (권장)

인터넷에 연결된 Pi에서 실행한다.

Raspberry Pi OS **Bookworm 이상**에서는 Python 가상환경이 먼저 필요하다.

```bash
sudo apt install python3-venv
python -m venv env --system-site-packages
source env/bin/activate
```

설치 스크립트 실행:

```bash
sudo apt install -y wget
pip3 install adafruit-python-shell
wget https://github.com/adafruit/Raspberry-Pi-Installer-Scripts/raw/main/i2samp.py
sudo -E env PATH=$PATH python3 i2samp.py
```

> 오프라인 대안: 이 저장소에 MIT 라이선스 확인 후 백업해 둔
> [`third_party/adafruit-raspberry-pi-installer-scripts/i2samp.py`](../third_party/adafruit-raspberry-pi-installer-scripts/i2samp.py)를
> `wget` 대신 사용할 수 있다. 출처·해시·라이선스는 같은 디렉터리의
> `README.md`와 `LICENSE`를 참고한다.

스크립트에는 재생 시작/종료 시 팝 노이즈를 줄이기 위해 유휴 시
무음(quiet audio)을 계속 재생하는 systemd 헬퍼 서비스가 포함되어 있다
(Pi Zero 기준 CPU 약 5%, Pi 2/3에서는 무시할 수준). RetroPie에서는
불필요하다.

이후 절차:

1. 설치 완료 후 **재부팅**한다.
2. 재부팅 후 **스크립트를 한 번 더 실행**한다. 스피커 테스트 여부를
   물으면 `y`를 입력하고 소리가 나는지 확인한다.
3. 소리가 심하게 왜곡되면 볼륨이 너무 큰 것일 수 있다. 데스크톱/
   RetroPie 볼륨 조절이 나타나려면 테스트 후 **한 번 더 재부팅**
   (`sudo reboot`)해야 한다.
4. `alsamixer`를 실행해 방향키로 볼륨을 낮춘다. **50%부터 시작**하는
   것이 좋다.
5. 그래도 문제가 있으면 스크립트를 다시 실행하고 `/dev/zero playback
   service`를 `N`(비활성화)으로 선택해 본다.

### 4.2 수동 설치

스크립트를 쓸 수 없을 때만 아래를 순서대로 진행한다.

**1) 블랙리스트 해제 (파일이 있을 때만)**

```bash
sudo nano /etc/modprobe.d/raspi-blacklist.conf
```

아래 줄이 있으면 앞에 `#`를 붙여 주석 처리한다.

```
# blacklist i2c-bcm2708
# blacklist snd-soc-pcm512x
# blacklist snd-soc-wm8804
```

**2) 헤드폰 오디오 모듈 비활성화**

```bash
sudo nano /etc/modules
```

`snd_bcm2835` 줄이 있으면 앞에 `#`를 붙인다.

**3) `/etc/asound.conf` 작성**

```bash
sudo nano /etc/asound.conf
```

```text
pcm.speakerbonnet {
   type hw card 0
}

pcm.dmixer {
   type dmix
   ipc_key 1024
   ipc_perm 0666
   slave {
     pcm "speakerbonnet"
     period_time 0
     period_size 1024
     buffer_size 8192
     rate 44100
     channels 2
   }
}

ctl.dmixer {
    type hw card 0
}

pcm.softvol {
    type softvol
    slave.pcm "dmixer"
    control.name "PCM"
    control.card 0
}

ctl.softvol {
    type hw card 0
}

pcm.!default {
    type             plug
    slave.pcm       "softvol"
}
```

**4) Device Tree 오버레이 추가**

```bash
sudo nano /boot/firmware/config.txt   # 구형 OS는 /boot/config.txt
```

`dtparam=audio=on` 줄이 있으면 앞에 `#`를 붙여 끄고, 파일 끝에 추가한다.

```text
dtoverlay=max98357a
```

저장 후 재부팅한다.

```bash
sudo reboot
```

## 5. 동작 테스트

```bash
# 화이트 노이즈 테스트 (좌/우 번갈아 출력)
# 모노 앰프는 좌우가 합쳐져 연속 노이즈로 들린다
speaker-test -c2

# WAV 테스트
speaker-test -c2 --test=wav -w /usr/share/sounds/alsa/Front_Center.wav

# MP3 스트리밍 테스트
sudo apt-get install -y mpg123
mpg123 http://ice1.somafm.com/u80s-128-mp3
```

참고: 오래된 Pi 커널은 I2S 모노 출력을 지원하지 않아 스테레오 재생만
가능하므로, 모노 파일은 스테레오로 변환해야 할 수 있다.

### 볼륨 조절

```bash
alsamixer   # 방향키 ↑/↓로 조절, ESC로 종료
```

MAX98357A 자체에는 소프트웨어 볼륨이 없어, 위 `asound.conf`의
`softvol` 플러그인이 소프트웨어 볼륨(`PCM`)을 제공한다. 데스크톱에서
볼륨 아이콘에 X가 표시되면 재부팅한다(설치 후 두 번 재부팅이 필요할
수 있다).

## 6. 트러블슈팅

| 증상 | 조치 |
|------|------|
| 소리가 전혀 안 남 | 배선(GPIO 18/19/21, 5V, GND) 확인 → `dtoverlay=max98357a` 적용·재부팅 확인 → 설치 스크립트 재실행 후 테스트에서 `y` 선택 |
| 소리가 심하게 왜곡됨 | `alsamixer`로 볼륨을 50% 이하로 낮춤. 전원 용량(800mA 이상) 확인 |
| 재생 시작/종료 시 팝·클릭 | §6.1의 dmix 설정 적용, 또는 설치 스크립트의 `/dev/zero playback service` 활성화 |
| 재생 중 계속 팝 발생 | 서비스가 방해되는 경우 스크립트 재실행 시 해당 서비스를 `N`으로 비활성화 |
| 볼륨 조절이 안 보임 | 재부팅 (데스크톱 인식에는 설치 후 2회 재부팅이 필요할 수 있음) |
| 데스크톱이 `~/.asoundrc`를 덮어씀 | Pi OS Desktop의 볼륨 위젯 문제. `chattr +i ~/.asoundrc`로 변경 금지 설정 (Lite에서는 해당 없음) |
| Docker `player`에서 `snd_pcm_dmix_open unable to open slave` | §6.2 참고. 컨테이너가 호스트 dmix에 합류하지 못하고 hw 장치를 직접 열려다 실패하는 경우와, 카드 번호/이름 불일치가 대표 원인 |

### 6.1 팝 노이즈 감소 설정 (Pi I2S Tweaks)

샘플레이트가 바뀔 때 비트 클럭이 변하면서 팝이 발생하는 경우,
dmix로 출력 샘플레이트를 고정하면 완화된다.

1. `/boot/firmware/config.txt`에 추가:

   ```text
   dtoverlay=i2s-mmap
   ```

2. `/etc/asound.conf`를 dmix 기본 출력으로 구성한다. 소프트웨어
   볼륨까지 필요하면 §4.2의 3)번 `asound.conf`(softvol 포함)를 그대로
   `~/.asoundrc`에 넣어도 된다.
3. 재부팅 → 테스트 오디오 1회 재생 → 재부팅하면 `alsamixer`와
   동기화된다.

이 설정 후에도 재생 맨 처음에는 가끔 팝이 한 번 발생할 수 있다.

### 6.2 Docker에서 `unable to open slave`가 뜰 때

```text
ALSA lib pcm_dmix.c:999:(snd_pcm_dmix_open) unable to open slave
```

dmix 플러그인이 그 아래 실제 hw 카드(`speakerbonnet`)를 열지 못했다는
뜻이다. 호스트에서는 소리가 나는데 컨테이너에서만 이 오류가 나면
아래 두 가지가 대표 원인이다.

1. **호스트 dmix와 컨테이너 dmix가 따로 논다.** dmix 공유는 SysV IPC
   (`ipc_key 1024`)로 이뤄지는데 Docker 컨테이너는 기본적으로 IPC
   네임스페이스가 분리돼 있다. 그러면 컨테이너 쪽 dmix가 hw 장치를
   직접 열려 하고, 호스트의 `/dev/zero` 재생 서비스(`aplay.service`)나
   호스트 dmix가 이미 장치를 잡고 있으면 `EBUSY`로 실패한다.
2. **카드 지정 불일치.** 컨테이너는 호스트 `/etc/asound.conf`를
   보지 못하므로, 기본값 `card 0`을 쓴다. I2S 카드가 0번이 아니면
   (예: HDMI가 0번, I2S는 `sndrpigooglevoi`) 엉뚱한 카드를 열게 된다.

조치는 `docker-compose.yml`의 `player` 서비스에 이미 반영돼 있다.

- `ipc: host` — 컨테이너가 호스트 dmix에 합류한다.
- `/etc/asound.conf:/etc/asound.conf:ro` 마운트 — 카드 이름·dmixer·
  softvol 정의를 호스트와 동일하게 쓴다.

Pi에서 확인할 순서:

```bash
# 1) 호스트에서 카드가 보이는지, 호스트 재생이 되는지 먼저 확인
aplay -l
cat /proc/asound/cards
speaker-test -c2

# 2) 장치를 누가 잡고 있는지, /dev/zero 서비스가 켜져 있는지 확인
systemctl is-active aplay.service
sudo fuser -v /dev/snd/*

# 3) 위 compose 수정 반영 후 재생성
docker compose up -d --force-recreate player
docker compose logs -f player
```

호스트의 `speaker-test`마저 실패하면 Docker 문제가 아니라 카드
인식 문제이므로 배선과 `dtoverlay`, 재부팅 여부를 먼저 본다.
반대로 `sudo systemctl stop aplay.service` 후 컨테이너 재생이 되면
1번 원인으로 확정된다. 이 경우 `ipc: host`를 유지한 채 서비스를
다시 켜도 되고, 팝 방지 서비스가 불필요하면 꺼 둬도 된다.

## 7. Si-Gnal 프로젝트에서의 사용

- 이 설정이 끝나면 앰프는 Pi의 **기본 ALSA 출력 장치**가 된다.
  Si-Gnal의 `cmd/player`와 `cmd/server`(`?play=speaker`)는
  oto/beep → ALSA 기본 장치로 재생하므로 **애플리케이션 코드 변경은
  필요 없다.**
- 설치 스크립트·`config.txt`·`asound.conf` 설정은 **호스트 OS**에서
  수행한다. 컨테이너 안에서는 수행할 수 없다.
- Docker 실행 시에는 `docker-compose.yml`처럼 호스트의 `/dev/snd`를
  플레이어 컨테이너에 마운트해야 한다. 컨테이너를 띄우기 전에
  호스트에서 `speaker-test -c2`로 먼저 검증한다.

  ```bash
  speaker-test -c2   # 호스트에서 소리 확인 후
  docker compose up -d
  ```

- Docker에서는 여기에 더해 **`ipc: host`와 호스트 `/etc/asound.conf`
  마운트**가 필요하다. 없으면 `snd_pcm_dmix_open unable to open slave`
  오류가 난다 (§6.2).

## 참고 링크

- [가이드 원문 (Raspberry Pi Usage)](https://learn.adafruit.com/adafruit-max98357-i2s-class-d-mono-amp/raspberry-pi-usage)
- [제품 페이지 — Adafruit I2S 3W Class D Amplifier Breakout (MAX98357A)](https://www.adafruit.com/product/3006)
- [가이드 전체 PDF](https://cdn-learn.adafruit.com/downloads/pdf/adafruit-max98357-i2s-class-d-mono-amp.pdf)
