# /// script
# dependencies = [
#   "requests",
#   "beautifulsoup4",
# ]
# ///

import requests
from bs4 import BeautifulSoup
import random
import time
import sys
import re

# Base configuration
BASE_URL = "https://www.poemlove.co.kr/bbs/board.php?bo_table=tb01"
HEADERS = {
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
    "Referer": "https://www.poemlove.co.kr/"
}

def get_last_page():
    """게시판의 마지막 페이지 번호를 가져옵니다."""
    try:
        response = requests.get(BASE_URL, headers=HEADERS, timeout=10)
        response.raise_for_status()
        soup = BeautifulSoup(response.text, 'html.parser')
        
        # '맨끝' 페이지 링크 (fa-angle-double-right 아이콘 포함)를 찾습니다.
        last_page_icon = soup.select_one('i.fa-angle-double-right')
        if last_page_icon and last_page_icon.parent and last_page_icon.parent.name == 'a':
            href = last_page_icon.parent.get('href', '')
            if 'page=' in href:
                return int(href.split('page=')[1].split('&')[0])
        
        # 다른 모든 페이지 링크에서 가장 큰 값을 찾습니다.
        import re
        pages = []
        for a in soup.select('a[href*="page="]'):
            href = a.get('href', '')
            match = re.search(r'page=(\d+)', href)
            if match:
                pages.append(int(match.group(1)))
                
        if pages:
            return max(pages)
            
        return 1
    except Exception as e:
        print(f"마지막 페이지 확인 중 오류 발생: {e}")
        return 1

def get_poem_links(page_num):
    """특정 페이지에서 시 링크(wr_id) 목록을 가져옵니다."""
    url = f"{BASE_URL}&page={page_num}"
    try:
        response = requests.get(url, headers=HEADERS, timeout=10)
        response.raise_for_status()
        soup = BeautifulSoup(response.text, 'html.parser')
        
        links = []
        for a in soup.select('a[href*="wr_id="]'):
            href = a.get('href', '')
            if 'bo_table=tb01' in href and 'wr_id=' in href:
                wr_id = href.split('wr_id=')[1].split('&')[0]
                if wr_id not in links:
                    links.append(wr_id)
        return links
    except Exception as e:
        print(f"페이지 {page_num} 읽기 오류: {e}")
        return []

def get_poem_detail(wr_id):
    """시 상세 내용을 가져옵니다."""
    url = f"{BASE_URL}&wr_id={wr_id}"
    try:
        response = requests.get(url, headers=HEADERS, timeout=10)
        response.raise_for_status()
        soup = BeautifulSoup(response.text, 'html.parser')
        
        title_tag = soup.select_one('h1')
        title = title_tag.get_text(strip=True) if title_tag else "제목 없음"
        
        author = "미상"
        for strong in soup.select('strong'):
            if strong.parent and '저자' in strong.parent.get_text():
                author = strong.get_text(strip=True)
                break
        
        content_tag = soup.select_one('.view-content')
        if content_tag:
            content = content_tag.get_text('\n')
            lines = [line.strip() for line in content.split('\n')]
            content = '\n'.join(lines)
            content = re.sub(r'\n{3,}', '\n\n', content).strip()
            
            # 본문에서 저자 찾기 시도
            if author == "미상" and "저자 :" in content:
                try:
                    author = content.split("저자 :")[1].split()[0].strip()
                except:
                    pass
        else:
            content = "내용을 불러올 수 없습니다."
            
        return {
            "title": title,
            "author": author,
            "content": content,
            "url": url
        }
    except Exception as e:
        print(f"상세 내용 가져오기 오류: {e}")
        return None

def main():
    print("게시판 정보를 확인 중입니다...", end="\r")
    last_page = get_last_page()
    print(f"전체 {last_page}개 페이지 중 무작위 선택 중...    ")
    
    # 무작위 페이지 선택
    random_page = random.randint(1, last_page)
    wr_ids = get_poem_links(random_page)
    
    if not wr_ids:
        print("시 목록을 가져오지 못했습니다.")
        return

    # 무작위 시 선택
    random_wr_id = random.choice(wr_ids)
    poem = get_poem_detail(random_wr_id)
    
    if poem:
        print("\n" + "="*50)
        print(f"📜 {poem['title']}")
        print(f"👤 저자: {poem['author']}")
        print("-" * 50)
        print(f"\n{poem['content']}\n")
        print("="*50)
        print(f"🔗 출처: {poem['url']}\n")
    else:
        print("시 내용을 불러오지 못했습니다.")

if __name__ == "__main__":
    main()
