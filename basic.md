# Couchbase Database Fundamentals

* Database/Bucket: royaltypool
* Schema/Scope: master
* Table/Collection: create your own

## How to create your Scope? (brand new scope)
> Syntax: CREATE SCOPE bucket.scopename;
  
> CREATE SCOPE `royaltypool`.`master`;
  
## How to create your Collection?
> CREATE COLLECTION bucket.scopename.collectionName;
  
> CREATE COLLECTION `royaltypool`.`master`.`client`;
  
## [How to create Index?](https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/createindex.html)

### Primary Named Index
> CREATE PRIMARY INDEX idx_default_primary ON `royalypool` USING GSI;

### Primary Normal Index
> CREATE PRIMARY INDEX ON keyspace_name USING GSI;

### Secondary Index
> CREATE INDEX `IndexName` ON `bucket`.`scope`.`collection`(`login_id`);

## Tutorial You may read:
* https://blog.couchbase.com/simplifying-query-index-with-collections/
* https://docs.couchbase.com/server/current/getting-started/try-a-query.html
