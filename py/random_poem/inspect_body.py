# /// script
# dependencies = ["requests", "beautifulsoup4"]
# ///
import requests
from bs4 import BeautifulSoup

url = 'https://www.poemlove.co.kr/bbs/board.php?bo_table=tb01&wr_id=48975'
r = requests.get(url, headers={'User-Agent': 'Mozilla'})
s = BeautifulSoup(r.text, 'html.parser')
c = s.select_one('.view-content')
print(c.prettify() if c else 'None')
