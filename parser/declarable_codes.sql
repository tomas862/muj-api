CREATE TABLE nomenclature_declarable_codes (
    id SERIAL PRIMARY KEY,
    nomenclature_id INTEGER NOT NULL REFERENCES nomenclatures(id) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    declarable_start_date DATE NOT NULL,
    is_leaf BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(nomenclature_id)
);

CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add index for nomenclature_id which will be used in joins
CREATE INDEX idx_nomenclature_declarable_codes_nomenclature_id 
ON nomenclature_declarable_codes(nomenclature_id);

CREATE TRIGGER update_nomenclatures_declarable_codes_modtime
BEFORE UPDATE ON nomenclature_declarable_codes
FOR EACH ROW EXECUTE FUNCTION update_modified_column();
