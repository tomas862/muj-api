CREATE EXTENSION IF NOT EXISTS ltree;

CREATE TABLE nomenclatures (
    id SERIAL PRIMARY KEY,
    goods_code VARCHAR(13) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    hierarchy_path LTREE NOT NULL,
    indent SMALLINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(goods_code)
);

CREATE TABLE nomenclature_descriptions (
    id SERIAL PRIMARY KEY,
    nomenclature_id INTEGER NOT NULL REFERENCES nomenclatures(id) ON DELETE CASCADE,
    language CHAR(2) NOT NULL,
    description TEXT NOT NULL,
    descr_start_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(nomenclature_id, language)
);

CREATE INDEX idx_nomenclatures_code ON nomenclatures(goods_code);
CREATE INDEX idx_nomenclature_descriptions_language ON nomenclature_descriptions(language);


CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER update_nomenclatures_modtime
BEFORE UPDATE ON nomenclatures
FOR EACH ROW EXECUTE FUNCTION update_modified_column();


CREATE TRIGGER update_nomenclature_descriptions_modtime
BEFORE UPDATE ON nomenclature_descriptions
FOR EACH ROW EXECUTE FUNCTION update_modified_column();