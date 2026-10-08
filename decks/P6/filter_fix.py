# Keep only the cards named on the command line (the P6 data fix of pm-tools-judge).
import json, sys
src, out, keep = sys.argv[1], sys.argv[2], set(sys.argv[3:])
cards = [c for c in json.load(open(src)) if c.get("customId", c.get("id")) in keep]
json.dump(cards, open(out, "w"), indent=2)
print(len(cards), "cards kept")
