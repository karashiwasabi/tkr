-- C:\Users\wasab\OneDrive\デスクトップ\TKR\schema.sql

CREATE TABLE IF NOT EXISTS product_master (
    product_code TEXT PRIMARY KEY,
    yj_code TEXT,
    gs1_code TEXT,
    product_name TEXT,
    kana_name TEXT,
    kana_name_short TEXT, -- ★ WASABIからの追加
    generic_name TEXT,    -- ★ WASABIからの追加
    maker_name TEXT,
    
specification 

TEXT,
    usage_classification TEXT,
    package_form TEXT,
    yj_unit_name TEXT,
    yj_pack_unit_qty REAL,
    jan_pack_inner_qty REAL,
    jan_unit_code INTEGER,
    jan_pack_unit_qty REAL,
    origin TEXT,
    nhi_price REAL,
    purchase_price REAL,
    flag_poison INTEGER,
    flag_deleterious INTEGER,
    flag_narcotic INTEGER,
    flag_psychotropic INTEGER,
    flag_stimulant INTEGER,
    flag_stimulant_raw INTEGER,
    is_order_stopped INTEGER DEFAULT 0,
    supplier_wholesale TEXT,
    group_code TEXT,
  
 

 shelf_number TEXT,
    category TEXT,
    user_notes TEXT
);
CREATE TABLE IF NOT EXISTS transaction_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  transaction_date TEXT,
  client_code TEXT,
  receipt_number TEXT,
  line_number TEXT,
  flag INTEGER,
  jan_code TEXT,
  yj_code TEXT,
  product_name TEXT,
  kana_name TEXT,
  usage_classification TEXT,
  package_form TEXT,
  package_spec TEXT,
  maker_name TEXT,
  dat_quantity REAL,
  jan_pack_inner_qty REAL,
  jan_quantity REAL,
  jan_pack_unit_qty REAL,
  jan_unit_name TEXT,
  jan_unit_code TEXT,
  yj_quantity REAL,
  yj_pack_unit_qty REAL,
  yj_unit_name TEXT,
  unit_price REAL,
  purchase_price REAL,
  supplier_wholesale TEXT,
  subtotal REAL,
  tax_amount REAL,
  
 tax_rate REAL, 
  

expiry_date TEXT,
  lot_number TEXT,
  flag_poison INTEGER,
  flag_deleterious INTEGER,
  flag_narcotic INTEGER,
  flag_psychotropic INTEGER,
  flag_stimulant INTEGER,
  flag_stimulant_raw INTEGER,
  process_flag_ma TEXT
);
CREATE TABLE IF NOT EXISTS jcshms (
  JC000 TEXT, JC001 TEXT, JC002 TEXT, JC003 TEXT, JC004 TEXT, JC005 TEXT, JC006 TEXT, JC007 TEXT, JC008 TEXT, JC009 TEXT,
  JC010 TEXT, JC011 TEXT, JC012 TEXT, JC013 TEXT, JC014 TEXT, JC015 TEXT, JC016 TEXT, JC017 TEXT, JC018 TEXT, JC019 TEXT,
  JC020 TEXT, JC021 TEXT, JC022 TEXT, JC023 TEXT, JC024 TEXT, JC025 TEXT, JC026 TEXT, JC027 TEXT, JC028 TEXT, JC029 TEXT,
  JC030 TEXT, JC031 TEXT, JC032 TEXT, JC033 TEXT, JC034 TEXT, JC035 TEXT, JC036 TEXT, JC037 TEXT, JC038 TEXT, JC039 TEXT,
  JC040 TEXT, JC041 TEXT, 
JC042 TEXT, JC043 TEXT, 

JC044 REAL, JC045 TEXT, JC046 TEXT, JC047 TEXT, JC048 TEXT, JC049 TEXT, 
  JC050 REAL, JC051 TEXT, JC052 TEXT, JC053 TEXT, JC054 TEXT, JC055 TEXT, JC056 TEXT, JC057 TEXT, JC058 TEXT, JC059 TEXT,
  JC060 TEXT, JC061 INTEGER, JC062 INTEGER, JC063 INTEGER, JC064 INTEGER, JC065 INTEGER, JC066 INTEGER, JC067 TEXT, JC068 TEXT, JC069 TEXT,
  JC070 TEXT, JC071 TEXT, JC072 TEXT, JC073 TEXT, JC074 TEXT, JC075 TEXT, JC076 TEXT, JC077 TEXT, JC078 TEXT, JC079 TEXT,
  JC080 TEXT, JC081 TEXT, JC082 TEXT, JC083 TEXT, JC084 TEXT, JC085 TEXT, JC086 TEXT, JC087 TEXT, JC088 TEXT, JC089 
TEXT,
  JC090 TEXT, 
JC091 TEXT, JC092 TEXT, JC093 TEXT, JC094 TEXT, JC095 TEXT, JC096 TEXT, JC097 TEXT, JC098 TEXT, JC099 TEXT, 
  JC100 TEXT, JC101 TEXT, JC102 TEXT, JC103 TEXT, JC104 TEXT, JC105 TEXT, JC106 TEXT, JC107 TEXT, JC108 TEXT, JC109 TEXT,
  JC110 TEXT, JC111 TEXT, JC112 TEXT, JC113 TEXT, JC114 TEXT, JC115 TEXT, JC116 TEXT, JC117 TEXT, JC118 TEXT, JC119 TEXT,
  JC120 TEXT, JC121 TEXT, JC122 TEXT, JC123 TEXT, JC124 REAL,
  PRIMARY KEY(JC000)
);
CREATE TABLE IF NOT EXISTS jancode (
  JA000 TEXT, JA001 TEXT, JA002 TEXT, JA003 TEXT, JA004 TEXT, JA005 TEXT, JA006 REAL, JA007 TEXT, JA008 REAL, JA009 TEXT,
  JA010 TEXT, JA011 TEXT, JA012 TEXT, JA013 TEXT, JA014 TEXT, JA015 TEXT, JA016 TEXT, JA017 TEXT, JA018 TEXT, JA019 TEXT,
  JA020 TEXT, JA021 TEXT, JA022 TEXT, JA023 TEXT, JA024 TEXT, JA025 TEXT, JA026 TEXT, JA027 TEXT, JA028 TEXT, JA029 TEXT,
  PRIMARY KEY(JA001)
);
CREATE INDEX IF NOT EXISTS idx_transactions_jan_code ON transaction_records (jan_code);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transaction_records (transaction_date);
CREATE INDEX IF NOT EXISTS idx_transactions_flag ON transaction_records (flag);
CREATE INDEX IF NOT EXISTS idx_product_master_kana_name ON product_master (kana_name);
CREATE INDEX IF NOT EXISTS idx_product_master_yj_code ON product_master (yj_code);
CREATE INDEX IF NOT EXISTS idx_jcshms_yj_code ON jcshms (JC009);
CREATE TABLE IF NOT EXISTS units (
    code TEXT PRIMARY KEY,
    name TEXT
);
CREATE TABLE IF NOT EXISTS medicine_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    yj_code TEXT UNIQUE,
    data TEXT
);
CREATE INDEX IF NOT EXISTS idx_medicine_data_yj_code ON medicine_data (yj_code);

CREATE TABLE IF NOT EXISTS price_revisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    revision_date TEXT NOT NULL,
    yj_code TEXT NOT NULL,
    old_price REAL,
    new_price REAL,
    UNIQUE(revision_date, yj_code)
);
CREATE TABLE IF NOT EXISTS price_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    yj_code TEXT NOT NULL,
    price REAL NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT,
    UNIQUE(yj_code, start_date)
);
CREATE INDEX IF NOT EXISTS idx_price_history_yj_code ON price_history (yj_code);
CREATE INDEX IF NOT EXISTS idx_price_history_start_date ON price_history (start_date);
CREATE INDEX IF NOT EXISTS idx_price_history_end_date ON price_history (end_date);

CREATE INDEX IF NOT EXISTS idx_product_master_gs1_code ON product_master (gs1_code);
CREATE INDEX IF NOT EXISTS idx_transactions_receipt_number ON transaction_records (receipt_number);

CREATE INDEX IF NOT EXISTS idx_transactions_flag_date ON transaction_records (flag, transaction_date);
-- ▼▼▼【ここから変更】backorders テーブルに jan_code を追加 ▼▼▼
CREATE TABLE IF NOT EXISTS backorders (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  order_date TEXT NOT NULL,
  jan_code TEXT,
  yj_code TEXT NOT NULL,
  product_name TEXT,
  package_form TEXT NOT NULL,
  jan_pack_inner_qty REAL NOT NULL,
  yj_unit_name TEXT NOT NULL,
  order_quantity REAL NOT NULL,
  remaining_quantity REAL NOT NULL,
  wholesaler_code TEXT,
  yj_pack_unit_qty REAL,
  jan_pack_unit_qty REAL,
  jan_unit_code INTEGER
);
-- ▲▲▲【変更ここまで】▲▲▲ 
CREATE TABLE IF NOT EXISTS precomp_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  transaction_date TEXT,
  client_code TEXT,
  receipt_number TEXT,
  line_number TEXT,
  jan_code TEXT,
  yj_code TEXT,
  product_name TEXT,
  kana_name TEXT,
  usage_classification TEXT,
  package_form TEXT,
  package_spec TEXT,
  maker_name TEXT,
  jan_pack_inner_qty REAL,
  jan_quantity REAL,
  jan_pack_unit_qty REAL,
  jan_unit_name TEXT,
  jan_unit_code TEXT,
  yj_quantity REAL,
  yj_pack_unit_qty REAL,
  yj_unit_name TEXT,
  purchase_price REAL,
  supplier_wholesale TEXT,
  created_at TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  UNIQUE(client_code, jan_code)
);
CREATE TABLE IF NOT EXISTS inventory_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inventory_date TEXT NOT NULL,
    product_code TEXT NOT NULL,
    yj_code TEXT,
    product_name TEXT,
    yj_quantity REAL NOT NULL,
    yj_unit_name TEXT,
    package_spec TEXT,
    UNIQUE(inventory_date, product_code)
);
CREATE INDEX IF NOT EXISTS idx_inventory_history_date_code ON inventory_history (inventory_date, product_code);
CREATE INDEX IF NOT EXISTS idx_inventory_history_code_date ON inventory_history (product_code, inventory_date);
CREATE TABLE IF NOT EXISTS client_master (
  client_code TEXT PRIMARY KEY,
  client_name TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS wholesalers (
  wholesaler_code TEXT PRIMARY KEY,
  wholesaler_name TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS code_sequences (
  name TEXT PRIMARY KEY,
  last_no INTEGER NOT NULL
);
INSERT OR IGNORE INTO code_sequences(name, last_no) VALUES ('MA2Y', 0);
INSERT OR IGNORE INTO code_sequences(name, last_no) VALUES ('CL', 0);
-- ▼▼▼【ここに追加】仮ProductCode用シーケンス ▼▼▼
INSERT OR IGNORE INTO code_sequences(name, last_no) VALUES ('MA2J', 0);
-- ▲▲▲【追加ここまで】▲▲▲

-- ▼▼▼【ここから追加】見積・価格設定機能用テーブル ▼▼▼
CREATE TABLE IF NOT EXISTS product_quotes (
    product_code TEXT NOT NULL,
    wholesaler_code TEXT NOT NULL,
    quote_price REAL NOT NULL,
    quote_date TEXT NOT NULL,
    PRIMARY KEY (product_code, wholesaler_code)
);
CREATE INDEX IF NOT EXISTS idx_product_quotes_product_code ON product_quotes (product_code);
CREATE INDEX IF NOT EXISTS idx_product_quotes_wholesaler_code ON product_quotes (wholesaler_code);
-- ▲▲▲【追加ここまで】▲▲▲

-- ▼▼▼【ここから追加】包装キー単位の在庫起点テーブル ▼▼▼
CREATE TABLE IF NOT EXISTS package_stock (
  package_key TEXT PRIMARY KEY,          -- 包装キー (例: '2647709N1060|包装小|5|Ｇ')
  yj_code TEXT,                          -- 対応するYJコード (検索用)
  stock_quantity_yj REAL NOT NULL,       -- その包装キーのYJ単位での在庫量
  last_inventory_date TEXT NOT NULL      -- この在庫量が確定した最新の棚卸日 (YYYYMMDD)
);
CREATE INDEX IF NOT EXISTS idx_package_stock_yj_code ON package_stock (yj_code);
-- ▲▲▲【追加ここまで】▲▲▲