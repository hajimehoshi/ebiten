#version 150

int modInt(int x, int y) {
	return x - y*(x/y);
}

ivec2 modInt(ivec2 x, int y) {
	return x - y*(x/y);
}

ivec3 modInt(ivec3 x, int y) {
	return x - y*(x/y);
}

ivec4 modInt(ivec4 x, int y) {
	return x - y*(x/y);
}

ivec2 modInt(ivec2 x, ivec2 y) {
	return x - y*(x/y);
}

ivec3 modInt(ivec3 x, ivec3 y) {
	return x - y*(x/y);
}

ivec4 modInt(ivec4 x, ivec4 y) {
	return x - y*(x/y);
}

int F0(void);

int F0(void) {
	int l0 = 0;
	l0 = 0;
	for (int l1 = 0; l1 < 10; l1++) {
		l0 = (l0) + (l1);
	}
	for (int l2 = 0; l2 < 4; l2++) {
		for (int l3 = 0; l3 < 4; l3++) {
			l0 = (l0) + ((l2) * (l3));
		}
	}
	for (int l3 = 0; l3 < 3; l3++) {
		l0 = (l0) + (1);
	}
	return l0;
}
