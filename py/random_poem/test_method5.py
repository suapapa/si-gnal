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

print("\n--- Method 5 ---")
# Extract text using newline separator, do not strip yet
text = c.get_text('\n')
# Strip spaces on each line and drop purely empty spaces making lines empty
lines = [line.strip() for line in text.split('\n')]
# Join with newlines
res = '\n'.join(lines)
# Reduce 3 or more newlines to 2 newlines (preserve single empty lines)
res = re.sub(r'\n{3,}', '\n\n', res).strip()
print(res)
