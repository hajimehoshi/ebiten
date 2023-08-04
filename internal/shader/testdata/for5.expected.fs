uniform float U0;
uniform float U1;
uniform float U2;

int F0(in int l0);
vec4 F1(in vec4 l0);

int F0(in int l0) {
	return l0;
}

vec4 F1(in vec4 l0) {
	int l1 = 0;
	int l3 = 0;
	l1 = 0;
	for (int l2 = 0; l2 < 10; l2++) {
		int l3 = 0;
		l3 = F0(l2);
		l1 = (l1) + (l3);
		for (int l4 = 0; l4 < 10; l4++) {
			int l5 = 0;
			l5 = F0(l4);
			l1 = (l1) + (l5);
		}
	}
	l3 = 0;
	l1 = (l1) + (l3);
	return vec4(float(l1));
}

void main(void) {
	fragColor = F1(gl_FragCoord);
}
