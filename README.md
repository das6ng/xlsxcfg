# xlscfg

config data from excel sheets

# Examples

1. Basic
```protobuf
message MemberSheetRow {
    int32 ID = 1;
    string Name = 2;
    string Address = 3;
}
message MemberSheet {
    repeated MemberSheetRow List = 1;
}
```
