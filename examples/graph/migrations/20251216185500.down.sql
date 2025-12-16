COMMENT ON SCHEMA public IS 'standard public schema';

REVOKE USAGE ON TYPE contract_relationship FROM PUBLIC;

REVOKE USAGE ON TYPE contract_attribute FROM PUBLIC;

REVOKE USAGE ON TYPE contract FROM PUBLIC;

DROP TABLE contract_attribute;

DROP TABLE contract;

DROP TABLE contract_relationship;

