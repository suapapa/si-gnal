# /// script
# dependencies = ["requests", "beautifulsoup4"]
# ///
import requests
from bs4 import BeautifulSoup
url = "https://www.poemlove.co.kr/bbs/board.php?bo_table=tb01&wr_id=274903"
headers = {'User-Agent': 'Mozilla/5.0'}
r = requests.get(url, headers=headers)
soup = BeautifulSoup(r.text, 'html.parser')
for h1 in soup.select('h1'):
    print('h1 class:', h1.get('class', []), h1.text.strip()[:50])
for title in soup.select('.title'):
    print('.title:', title.text.strip()[:50])
for t in soup.select('div[class*="title"]'):
    print('div class title:', t.get('class', []), t.text.strip()[:50])
print('HTML Title: ', soup.title.string)
for con in soup.select('div[class*="con"]'):
    print('div class con:', con.get('class', []), con.text.strip()[:50])
for b in soup.select('div[class*="content"]'):
    print('div class content:', b.get('class', []), b.text.strip()[:50])
for b in soup.select('div[id*="con"]'):
    print('div id con:', b.get('id'), b.get('class', []), b.text.strip()[:50])
