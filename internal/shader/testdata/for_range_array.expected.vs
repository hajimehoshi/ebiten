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

uniform vec2 U0[4];

float F0(void);

float F0(void) {
	float l0[3];
	l0[0] = float(0);
	l0[1] = float(0);
	l0[2] = float(0);
	float l1 = float(0);
	float l5 = float(0);
	vec2 l7 = vec2(0);
	l1 = 0.0;
	for (int l2 = 0; l2 < 3; l2++) {
		l1 = (l1) + (1.0);
	}
	for (int l3 = 0; l3 < 3; l3++) {
		l1 = (l1) + ((l0)[l3]);
	}
	for (int l4 = 0; l4 < 3; l4++) {
		l5 = (l0)[l4];
		l1 = (l1) + ((float(l4)) * (l5));
	}
	for (int l6 = 0; l6 < 4; l6++) {
		l7 = (U0)[l6];
		l1 = (l1) + ((l7).x);
	}
	return l1;
}
