# /// script
# dependencies = ["requests", "beautifulsoup4"]
# ///
import requests
from bs4 import BeautifulSoup
url = "https://www.poemlove.co.kr/bbs/board.php?bo_table=tb01&wr_id=274903"
headers = {'User-Agent': 'Mozilla/5.0'}
r = requests.get(url, headers=headers)
soup = BeautifulSoup(r.text, 'html.parser')
title = soup.select_one('#bo_v_title')
print("Title:", title.get_text() if title else None)
con = soup.select_one('#bo_v_con')
print("Content:", con.get_text()[:100] if con else None)
