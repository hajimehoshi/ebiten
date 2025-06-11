float F0(void);
void F1(out float l0, out vec2 l1);
void F2(in bool l0, out float l1, out vec2 l2);

float F0(void) {
	float l0 = float(0);
	{
		l0 = 0.0;
	}
	return l0;
}

void F1(out float l0, out vec2 l1) {
	float l2 = float(0);
	vec2 l3 = vec2(0);
	{
		float l4 = float(0);
		vec2 l5 = vec2(0);
		l4 = 0.0;
		l5 = vec2(0.0);
		l2 = l4;
		l3 = l5;
	}
	l0 = l2;
	l1 = l3;
	return;
}

void F2(in bool l0, out float l1, out vec2 l2) {
	float l3 = float(0);
	vec2 l4 = vec2(0);
	{
		float l5 = float(0);
		vec2 l6 = vec2(0);
		l5 = 0.0;
		l6 = vec2(0.0);
		l3 = l5;
		l4 = l6;
	}
	l1 = l3;
	l2 = l4;
	return;
}
