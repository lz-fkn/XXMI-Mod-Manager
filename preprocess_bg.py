import sys
from PIL import Image, ImageFilter, ImageEnhance

INPUT_FILE = sys.argv[1]
OUTPUT_FILE = "frontend/assets/images/" + sys.argv[2]
TARGET_WIDTH = 1200
TARGET_HEIGHT = 800

def process_image():
    try:
        img = Image.open(INPUT_FILE)
    except FileNotFoundError:
        print(f"Error: {INPUT_FILE} not found. Please place an image named '{INPUT_FILE}' in the root.")
        return

    img_ratio = img.width / img.height
    target_ratio = TARGET_WIDTH / TARGET_HEIGHT

    if img_ratio > target_ratio:
        new_height = TARGET_HEIGHT
        new_width = int(new_height * img_ratio)
    else:
        new_width = TARGET_WIDTH
        new_height = int(new_width / img_ratio)

    img = img.resize((new_width, new_height), Image.Resampling.LANCZOS)

    left = (new_width - TARGET_WIDTH) / 2
    top = (new_height - TARGET_HEIGHT) / 2
    right = (new_width + TARGET_WIDTH) / 2
    bottom = (new_height + TARGET_HEIGHT) / 2
    img = img.crop((left, top, right, bottom))

    img = img.filter(ImageFilter.GaussianBlur(radius=3))

    enhancer = ImageEnhance.Brightness(img)
    img = enhancer.enhance(0.4)

    import os
    os.makedirs(os.path.dirname(OUTPUT_FILE), exist_ok=True)
    img.save(OUTPUT_FILE, "JPEG", quality=90)
    print(f"Background processed and saved to {OUTPUT_FILE}")

if __name__ == "__main__":
    process_image()