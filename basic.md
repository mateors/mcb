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

## Couchbase Multiple Data Insert

```
  INSERT INTO `travel-sample`.inventory.airline (KEY,VALUE)
VALUES ( "airline_4444",
    { "callsign": "MY-AIR",
      "country": "United States",
      "iata": "Z1",
      "icao": "AQZ",
      "name": "80-My Air",
      "id": "4444",
      "type": "airline"} ),
VALUES ( "airline_4445",
    { "callsign": "AIR-X",
      "country": "United States",
      "iata": "X1",
      "icao": "ARX",
      "name": "10-AirX",
      "id": "4445",
      "type": "airline"} )
RETURNING *;
```

## Tutorial You may read:
* https://blog.couchbase.com/simplifying-query-index-with-collections/
* https://docs.couchbase.com/server/current/getting-started/try-a-query.html
* https://docs.couchbase.com/files/Couchbase-N1QL-CheatSheet.pdf
* [Interactive N1QL Query playground] (https://query-tutorial.couchbase.com/tutorial/#1)
