void F0(in vec2 l0, out vec2 l1);

void F0(in vec2 l0, out vec2 l1) {
	vec2 l2 = vec2(0);
	l2 = vec2(0.0);
	{
		vec2 l3 = vec2(0);
		l3 = vec2(1.0);
		if (((l3).x) == (1.0)) {
			l1 = l3;
			return;
		}
	}
	l1 = l2;
	return;
}
