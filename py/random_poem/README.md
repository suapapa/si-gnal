# 무작위 시 읽기 (poemlove.co.kr)

이 프로젝트는 [시 사랑](https://www.poemlove.co.kr/bbs/board.php?bo_table=tb01) 게시판에서 무작위로 한 편의 시를 골라 터미널 화면에 출력해주는 파이썬 스크립트입니다.

## 🚀 빠른 시작 (uv 사용)

[uv](https://github.com/astral-sh/uv)는 초고속 파이썬 패키지 관리자입니다. 별도의 가상 환경 구축 없이도 다음 명령어로 즉시 실행할 수 있습니다.

### 1. uv 설치 (이미 설치된 경우 생략)
```bash
# macOS/Linux
curl -LsSf https://astral.sh/uv/install.sh | sh

# Windows (PowerShell)
powershell -c "irm https://astral.sh/uv/install.ps1 | iex"
```

### 2. 무작위 시 한 편 읽기
의존성(`requests`, `beautifulsoup4`)이 자동으로 관리되므로, 아래 명령어 하나만 입력하면 됩니다. 
실행하면 전체 게시판에서 무작위로 한 편의 시를 골라 화면에 출력합니다.

```bash
uv run random_poem.py
```

## ⚠️ 주의 사항
- **저작권**: 출력된 시 데이터는 개인적인 감상 및 학습용으로만 사용하시기 바랍니다.
