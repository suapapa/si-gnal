# /// script
# dependencies = ["requests", "beautifulsoup4"]
# ///
import requests
from bs4 import BeautifulSoup
import re

url = "https://www.poemlove.co.kr/bbs/board.php?bo_table=tb01&wr_id=575"
headers = {'User-Agent': 'Mozilla/5.0'}
r = requests.get(url, headers=headers)
soup = BeautifulSoup(r.text, 'html.parser')
c = soup.select_one('.view-content')

# Method 4 (Fix Method 2 safely)
print("\n--- Method 4 ---")
# To avoid tree modification issues during iteration, collect them first, or just use strings.
# The safest way is to use get_text with a special separator we know doesn't exist, 
# then replace it, but `get_text` also extracts text.
# Actually, the problem with `replace_with` was just mutating while iterating? No, `find_all` returns a list.
s4 = BeautifulSoup(str(c), 'html.parser')
c4 = s4.select_one('.view-content')
for br in c4.find_all('br'):
    br.replace_with('\n')
# Wait, let's see why it failed.
text = c4.get_text()
lines = [line.strip() for line in text.split('\n')]
res = '\n'.join(lines)
res = re.sub(r'\n{3,}', '\n\n', res).strip()
print(res)
