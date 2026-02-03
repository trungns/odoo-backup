import os
import requests
from datetime import datetime, timedelta
from pydrive.auth import GoogleAuth
from pydrive.drive import GoogleDrive
import configparser

# --------- ĐỌC CẤU HÌNH TỪ FILE ---------
def load_config():
    """Đọc cấu hình từ file config.ini"""
    config = configparser.ConfigParser()
    config_file = "config.ini"

    if not os.path.exists(config_file):
        print(f"CẢNH BÁO: Không tìm thấy file {config_file}!")
        print("Vui lòng tạo file config.ini với các thông tin cấu hình.")
        raise FileNotFoundError(f"File {config_file} không tồn tại")

    config.read(config_file, encoding='utf-8-sig')
    return config

# Đọc cấu hình
config = load_config()

# --------- CẤU HÌNH TỪ FILE ---------
ODOO_URL = config.get('ODOO', 'ODOO_URL')
ODOO_DB = config.get('ODOO', 'ODOO_DB')
ODOO_USERNAME = config.get('ODOO', 'ODOO_USERNAME')
ODOO_PASSWORD = config.get('ODOO', 'ODOO_PASSWORD')
MASTER_PASSWORD = config.get('ODOO', 'MASTER_PASSWORD')
DATABASE_NAME = config.get('ODOO', 'DATABASE_NAME')

BACKUP_DIR = config.get('BACKUP', 'BACKUP_DIR')
KEEP_DAYS = config.getint('BACKUP', 'KEEP_DAYS')
BACKUP_FORMAT = config.get('BACKUP', 'BACKUP_FORMAT')

GOOGLE_CREDENTIALS_FILE = config.get('GOOGLE_DRIVE', 'GOOGLE_CREDENTIALS_FILE')

# --------- XÁC THỰC PYDRIVE ---------
def authenticate_google_drive():
    """Xác thực với Google Drive."""
    gauth = GoogleAuth()
    gauth.LoadCredentialsFile("credentials.json")
    if gauth.credentials is None:
        gauth.LocalWebserverAuth()  # Cấp quyền nếu chưa có
    elif gauth.access_token_expired:
        gauth.Refresh()  # Làm mới nếu hết hạn
    else:
        gauth.Authorize()  # Xác thực
    gauth.SaveCredentialsFile("credentials.json")
    return GoogleDrive(gauth)

# --------- TẠO BẢN SAO LƯU ODOO ---------
def backup_odoo_database():
    """Kết nối tới Odoo và sao lưu cơ sở dữ liệu."""
    print("Bắt đầu sao lưu cơ sở dữ liệu...")
    url = f"{ODOO_URL}/web/database/backup"
    params = {
        "master_pwd": MASTER_PASSWORD,
        "name": DATABASE_NAME,
        "backup_format": BACKUP_FORMAT,
    }
    
    response = requests.post(url, data=params, stream=True)
    if response.status_code == 200:
        # Lưu file backup
        backup_filename = f"{ODOO_DB}_{datetime.now().strftime('%Y%m%d_%H%M%S')}.zip"
        backup_path = os.path.join(BACKUP_DIR, backup_filename)
        with open(backup_path, "wb") as f:
            for chunk in response.iter_content(chunk_size=1024):
                f.write(chunk)
        print(f"Đã tạo bản sao lưu: {backup_path}")
        return backup_path
    else:
        print("Không thể sao lưu cơ sở dữ liệu. Vui lòng kiểm tra cấu hình.")
        print("Chi tiết lỗi:", response.text)
        return None

# --------- TẢI LÊN GOOGLE DRIVE ---------
def upload_to_google_drive(drive, file_path):
    """Tải file lên Google Drive."""
    print(f"Đang tải lên Google Drive: {file_path}")
    file_name = os.path.basename(file_path)
    file_drive = drive.CreateFile({"title": file_name})
    file_drive.SetContentFile(file_path)
    file_drive.Upload()
    print(f"Đã tải lên thành công: {file_name}")

# --------- DỌN DẸP FILE BACKUP CŨ ---------
def clean_old_backups():
    """Xóa các file backup cũ."""
    print(f"Đang dọn dẹp các file backup cũ hơn {KEEP_DAYS} ngày...")
    now = datetime.now()
    for filename in os.listdir(BACKUP_DIR):
        file_path = os.path.join(BACKUP_DIR, filename)
        if os.path.isfile(file_path):
            file_time = datetime.fromtimestamp(os.path.getmtime(file_path))
            if now - file_time > timedelta(days=KEEP_DAYS):
                os.remove(file_path)
                print(f"Đã xóa file backup cũ: {file_path}")

# --------- CHƯƠNG TRÌNH CHÍNH ---------
def main():
    """Chạy chương trình backup."""
    if not os.path.exists(BACKUP_DIR):
        os.makedirs(BACKUP_DIR)

    # Xác thực Google Drive
    drive = authenticate_google_drive()

    # 1. Sao lưu cơ sở dữ liệu
    backup_path = backup_odoo_database()
    if not backup_path:
        return

    # 2. Tải file lên Google Drive
    upload_to_google_drive(drive, backup_path)

    # 3. Dọn dẹp file backup cũ
    clean_old_backups()

if __name__ == "__main__":
    main()
