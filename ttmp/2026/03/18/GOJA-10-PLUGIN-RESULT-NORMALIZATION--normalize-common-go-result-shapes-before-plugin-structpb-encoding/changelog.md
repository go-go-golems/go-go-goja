# Changelog

## 2026-03-18

Created GOJA-10 to address the remaining SDK ergonomics gap around result encoding. The ticket is intentionally narrow: normalize common Go slice/map shapes before `structpb` conversion without expanding the transport contract or introducing a reflection-heavy serializer.

## 2026-03-18

- Initial workspace created
