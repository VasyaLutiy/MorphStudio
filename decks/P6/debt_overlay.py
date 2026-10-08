# The P6 debt deck: pm-tools-judge's acceptance without the generation-2 overlay that blanks mcpserver/session.go
# (accepted in run 20261008-200807; mount.go calls SessionServer, so the blanked file breaks every build).
import json, sys
src, out = sys.argv[1], sys.argv[2]
cards = json.load(open(src))
old = '{"Replace":{"mcpserver/session.go":""}}'
for c in cards:
    assert c["acceptance"].count(old) == 1, c["customId"]
    c["acceptance"] = c["acceptance"].replace(old, '{"Replace":{}}')
json.dump(cards, open(out, "w"), indent=2)
print(len(cards), "cards, overlay emptied")
