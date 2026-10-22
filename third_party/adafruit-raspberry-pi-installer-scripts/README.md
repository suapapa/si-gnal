# 백업: Adafruit `i2samp.py`

이 디렉터리는 MAX98357A I2S 앰프 설치 스크립트 `i2samp.py`의 **수정 없는
백업본**이다. 업스트림이 사라지거나 네트워크 없는 환경에서도 Raspberry Pi를
설정할 수 있도록 보관한다. 사용법은
[docs/max98357a-raspberry-pi.md](../../docs/max98357a-raspberry-pi.md)를
참고한다.

## 출처 (Provenance)

| 항목 | 값 |
|------|----|
| 원본 파일 | <https://github.com/adafruit/Raspberry-Pi-Installer-Scripts/raw/main/i2samp.py> |
| 업스트림 저장소 | <https://github.com/adafruit/Raspberry-Pi-Installer-Scripts> |
| 파일 최종 커밋 | `cb127962b370b1261383e7254f436485d4778c13` (2026-07-14, "Soften i2samp aplay service dependency") |
| 백업일 | 2026-10-22 |
| SHA-256 | `95f75617ac0909291151f93d7d4703f693670c3fd706cebbea2ad1bb1565ed4a` |
| 무결성 확인 | `shasum -a 256 i2samp.py` 결과가 위 값과 같아야 한다 |

## 라이선스

문제 없음 — **MIT 라이선스**로 재배포·수정·복사가 허용된다.

- 파일 첫머리에 SPDX 표시가 있다:
  `SPDX-FileCopyrightText: 2020 Melissa LeBlanc-Williams for Adafruit Industries`
  / `SPDX-License-Identifier: MIT`
- 업스트림 저장소에도 `LICENSES/MIT.txt`가 포함되어 있다
  (루트 `LICENSE` 파일이 없어 GitHub 라이선스 API는 404를 반환하지만,
  파일 자체의 SPDX 표시와 `LICENSES/` 디렉터리로 라이선스가 명시된다).
- MIT 조건에 따라 저작권·허가 표시를 유지해야 하므로, 원본 파일은
  헤더를 포함해 **한 글자도 바꾸지 않았고**, 라이선스 전문을 옆의
  [`LICENSE`](LICENSE) 파일에 함께 둔다.
- 이 백업을 수정해서 쓸 경우에도 위 저작권 표시와 `LICENSE`를 유지해야
  한다.

## 실행 방법 (백업본 사용 시)

업스트림에서 받는 대신 이 디렉터리의 파일로 대체하면 된다. 의존성
`adafruit-python-shell` 설치는 여전히 필요하다.

```bash
sudo apt install -y python3-venv
python -m venv env --system-site-packages
source env/bin/activate
pip3 install adafruit-python-shell
sudo -E env PATH=$PATH python3 third_party/adafruit-raspberry-pi-installer-scripts/i2samp.py
```

스크립트는 Raspberry Pi에서 root 권한으로 실행되며, 실행 중
Install/Uninstall을 선택하는 대화형 메뉴가 표시된다.
