-- Main nomenclature items table
CREATE TABLE nomenclature_items (
    id SERIAL PRIMARY KEY,
    goods_code VARCHAR(20) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    hier_pos SMALLINT NOT NULL, -- Changed to SMALLINT for small integers
    indent SMALLINT NOT NULL,   -- Changed to SMALLINT for small integers
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Updated unique constraint to include end_date
    UNIQUE(goods_code, start_date, end_date)
);

-- Language-specific descriptions table
CREATE TABLE nomenclature_descriptions (
    id SERIAL PRIMARY KEY,
    nomenclature_item_id INTEGER NOT NULL REFERENCES nomenclature_items(id) ON DELETE CASCADE,
    language CHAR(2) NOT NULL,
    description TEXT NOT NULL,
    descr_start_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(nomenclature_item_id, language)
);

-- Indexes for common queries
CREATE INDEX idx_nomenclature_items_goods_code ON nomenclature_items(goods_code);
CREATE INDEX idx_nomenclature_items_start_date ON nomenclature_items(start_date);
CREATE INDEX idx_nomenclature_items_end_date ON nomenclature_items(end_date);
CREATE INDEX idx_nomenclature_items_hier_pos ON nomenclature_items(hier_pos);
CREATE INDEX idx_nomenclature_descriptions_language ON nomenclature_descriptions(language);
CREATE INDEX idx_nomenclature_descriptions_descr_start_date ON nomenclature_descriptions(descr_start_date);

-- Trigger to update the updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_nomenclature_items_modtime
BEFORE UPDATE ON nomenclature_items
FOR EACH ROW EXECUTE FUNCTION update_modified_column();

CREATE TRIGGER update_nomenclature_descriptions_modtime
BEFORE UPDATE ON nomenclature_descriptions
FOR EACH ROW EXECUTE FUNCTION update_modified_column();