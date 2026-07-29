package main

// selVTaps is the V-plane template produced by `lab wallsel renders4k/hd_00011121.png 20`: greedy forward selection on top of caeBytes' ten taps, at 3840x2160 and 11,121 regions.
// It is frozen here so it can be priced at every other operating point without reselecting, which is the only way to tell a real gain from a template fitted to one partition.
// The first ten entries are baseVTaps unchanged, so any prefix of length >= 10 is a strict superset of the published coder.
var selVTaps = []tap{
	{0, -1, 0}, {0, -2, 0}, {0, 0, -1}, {0, -1, -1}, {0, 1, -1},
	{0, 2, -1}, {0, 0, -2}, {0, -1, -2}, {0, 1, -2}, {0, -2, -1},
	{0, -3, 0}, {0, -3, -1}, {0, -4, 0}, {0, -4, -1}, {0, 3, -1},
	{0, -2, -2}, {0, 2, -2}, {0, -5, 0}, {0, -3, -2}, {0, 4, -1},
}

// The Hz plane keeps its ten baseline taps at every width: greedy selection was run on it too and every added bit made it worse, because Hz is determined by V except at junctions and the three taps that determine it are already there.
var selHTaps = baseHTaps
