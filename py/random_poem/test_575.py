# /// script
# dependencies = ["requests", "beautifulsoup4"]
# ///
import random_poem
poem = random_poem.get_poem_detail("575")
print("\n" + "="*50)
print(f"📜 {poem['title']}")
print(f"👤 저자: {poem['author']}")
print("-" * 50)
print(f"\n{poem['content']}\n")
print("="*50)
