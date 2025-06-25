import time
import threading
import requests
import supabase
import os
from selenium import webdriver
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from webdriver_manager.chrome import ChromeDriverManager
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor
import random

# Get configuration from environment variables
SUPABASE_URL = os.environ.get("SUPABASE_URL", "https://orzdjptliwwwhguldfok.supabase.co")
SUPABASE_KEY = os.environ.get("SUPABASE_KEY", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Im9yemRqcHRsaXd3d2hndWxkZm9rIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDAxNjQ5MzgsImV4cCI6MjA1NTc0MDkzOH0.yPZVODSO_8l0bglIfChr63KLO-SdLwDMNedEZT05VME")
DISCORD_WEBHOOK = os.environ.get("DISCORD_WEBHOOK", "https://discord.com/api/webhooks/1342555322929381441/RpVOyAfycPeObkR8038D0WyWYPQRCx0yjZexhYIxd8-5J6HPuUqEUqDQ0j4fr0-hzE3I")

# Set the scan interval to 1 hour (in seconds)
SCAN_INTERVAL = 3600

def get_supabase_client():
    return supabase.create_client(SUPABASE_URL, SUPABASE_KEY)

def retry_api_call(func, max_retries=5):
    for attempt in range(max_retries):
        try:
            return func()
        except Exception as e:
            wait_time = 2 ** attempt + random.uniform(0, 1)
            print(f"API call failed: {e}, retrying in {wait_time:.2f} seconds...")
            time.sleep(wait_time)
    print("Max retries reached. Skipping operation.")
    return None

def setup_driver():
    options = Options()
    options.add_argument("--headless")
    options.add_argument("--no-sandbox")
    options.add_argument("--disable-dev-shm-usage")
    service = Service(ChromeDriverManager().install())
    return webdriver.Chrome(service=service, options=options)

def check_stock(url):
    driver = setup_driver()
    driver.get(url)
    in_stock, img_url, price_yen = False, "", ""

    try:
        WebDriverWait(driver, 5).until(EC.presence_of_element_located((By.CLASS_NAME, "product-form__add-button")))
        add_buttons = driver.find_elements(By.CLASS_NAME, "product-form__add-button")
        in_stock = any("button--disabled" not in button.get_attribute("class") for button in add_buttons)
        img_element = driver.find_element(By.CLASS_NAME, "product-gallery__image")
        img_url = img_element.get_attribute("data-zoom") or img_element.get_attribute("data-srcset").split(" ")[0]
        price_element = driver.find_element(By.CSS_SELECTOR, ".product-form__info-content .price")
        if price_element:
            price_yen = price_element.text.replace("¥", "").replace("販売価格", "").strip()
    except Exception as e:
        print(f"Error processing {url}: {e}")
    
    driver.quit()
    return in_stock, img_url, price_yen

def process_entry(entry):
    url = entry["url"]
    prev_status = entry["in_stock"]
    in_stock, img_url, price_yen = check_stock(url)
    sb = get_supabase_client()

    def update_db():
        return sb.table("tracker").update({
            "in_stock": in_stock,
            "image": img_url,
            "price_yen": price_yen,
            "last_updated": str(datetime.now())
        }).eq("url", url).execute()
    
    retry_api_call(update_db)
    
    if in_stock != prev_status:
        message = {
            "content": f"🎉 **Item Restocked!**\n💴 Price: ¥{price_yen}\n🔗 {url}" if in_stock else f"⚠️ **Item Out of Stock!**\n🔗 {url}"
        }
        requests.post(DISCORD_WEBHOOK, json=message)

def update_stock():
    while True:
        print(f"Starting hourly scan at {datetime.now()}")
        sb = get_supabase_client()
        def fetch_entries():
            return sb.table("tracker").select("*").execute().data
        entries = retry_api_call(fetch_entries) or []
        
        print(f"Found {len(entries)} items to check")
        with ThreadPoolExecutor(max_workers=5) as executor:
            executor.map(process_entry, entries)
        
        print(f"Scan completed at {datetime.now()}. Next scan in 1 hour.")
        time.sleep(SCAN_INTERVAL)  # Sleep for 1 hour

def monitor_new_links():
    processed_urls = set()
    while True:
        sb = get_supabase_client()
        def fetch_entries():
            return sb.table("tracker").select("*").execute().data
        new_entries = retry_api_call(fetch_entries) or []
        for entry in new_entries:
            if entry["url"] not in processed_urls:
                process_entry(entry)
                processed_urls.add(entry["url"])
        time.sleep(60)  # Check for new links every minute

if __name__ == "__main__":
    print(f"Starting stock tracking application at {datetime.now()}")
    print(f"Scan interval set to {SCAN_INTERVAL/60} minutes")
    
    threading.Thread(target=update_stock, daemon=True).start()
    threading.Thread(target=monitor_new_links, daemon=True).start()
    
    while True:
        time.sleep(1)