# Odoo Backup Tool - Sao lưu tự động Odoo lên Google Drive

## Tính năng
- ✅ Tự động backup database Odoo
- ✅ Upload backup lên Google Drive
- ✅ Tự động xóa các file backup cũ
- ✅ Cấu hình bằng file config.ini (không cần build lại)
- ✅ Build thành file .exe để chạy trên Windows Server

## Yêu cầu hệ thống
- Python 3.7 trở lên (chỉ cần khi build)
- Windows Server (khi chạy file .exe)
- Kết nối đến Odoo server
- Google Drive API credentials

## Hướng dẫn build thành file .exe

### Cách 1: Build tự động (Đơn giản)
1. Mở Command Prompt với quyền Administrator
2. Di chuyển đến thư mục project:
   ```
   cd đường\dẫn\đến\odoo_backup
   ```
3. Chạy file build:
   ```
   build.bat
   ```
4. Đợi quá trình build hoàn tất
5. File .exe sẽ nằm trong thư mục `dist\odoo_backup.exe`

### Cách 2: Build thủ công
```bash
pip install -r requirements.txt
pyinstaller --onefile --console --name odoo_backup odoo_backup.py
```

## Cấu hình

Sau khi build, chỉnh sửa file `config.ini`:

```ini
[ODOO]
ODOO_URL = http://your-server:8069
DATABASE_NAME = your_database_name
MASTER_PASSWORD = your_master_password

[BACKUP]
BACKUP_DIR = backups
KEEP_DAYS = 5
BACKUP_FORMAT = zip

[GOOGLE_DRIVE]
GOOGLE_CREDENTIALS_FILE = client_secrets.json
```

## Triển khai lên Windows Server

1. Copy các file sau lên server:
   - `odoo_backup.exe`
   - `config.ini`
   - `credentials.json`
   - `client_secrets.json`

2. Chỉnh sửa `config.ini` với thông tin của bạn

3. Test chạy thử:
   ```
   odoo_backup.exe
   ```

## Tự động hóa với Windows Task Scheduler

1. Mở **Task Scheduler**
2. Chọn **Create Basic Task**
3. Đặt tên: "Odoo Daily Backup"
4. Trigger: Daily (hàng ngày)
5. Time: 02:00 AM
6. Action: **Start a program**
7. Program/script: `C:\path\to\odoo_backup.exe`
8. Start in: `C:\path\to\` (thư mục chứa exe)
9. Nhấn **Finish**

## Cấu trúc file

```
odoo_backup/
├── odoo_backup.py          # Code chính
├── config.ini              # File cấu hình
├── config.ini.example      # File mẫu
├── requirements.txt        # Thư viện Python
├── build.bat               # Script build tự động
├── build_simple.bat        # Script build đơn giản
├── HUONG_DAN_BUILD.txt     # Hướng dẫn chi tiết
├── README.md               # File này
├── credentials.json        # Google Drive credentials
└── client_secrets.json     # Google API secrets
```

## Xử lý lỗi thường gặp

### Lỗi: "File config.ini không tồn tại"
- Đảm bảo file `config.ini` nằm cùng thư mục với file .exe

### Lỗi: "Không thể sao lưu cơ sở dữ liệu"
- Kiểm tra ODOO_URL có đúng không
- Kiểm tra MASTER_PASSWORD có đúng không
- Kiểm tra DATABASE_NAME có tồn tại trong Odoo không

### Lỗi Google Drive authentication
- Chạy lại chương trình để xác thực lại
- Kiểm tra file `credentials.json` và `client_secrets.json`

## Cập nhật chương trình

Khi cần cập nhật code:
1. Sửa file `odoo_backup.py`
2. Chạy lại `build.bat`
3. Copy file mới trong `dist\` lên server
4. **Không cần build lại** nếu chỉ thay đổi cấu hình trong `config.ini`

## Lưu ý bảo mật

- ⚠️ File `config.ini` chứa password, cần bảo mật
- ⚠️ File `credentials.json` là thông tin nhạy cảm
- ⚠️ Không commit các file trên lên Git
- ✅ Sử dụng file `config.ini.example` để chia sẻ cấu trúc

## Hỗ trợ

Nếu gặp vấn đề, kiểm tra:
1. Log trong terminal khi chạy
2. Kết nối mạng đến Odoo server
3. Quyền truy cập Google Drive
4. Dung lượng ổ đĩa còn đủ không

---
**Version:** 2.0
**Last Updated:** 2026-02-02
