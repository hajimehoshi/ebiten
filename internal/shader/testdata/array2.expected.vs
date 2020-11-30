void F0(out vec2 l0[3]);

void F0(out vec2 l0[3]) {
	vec2 l1[2];
	l1[0] = vec2(0);
	l1[1] = vec2(0);
	vec2 l2[3];
	l2[0] = vec2(0);
	l2[1] = vec2(0);
	l2[2] = vec2(0);
	{
		vec2 l2[2];
		l2[0] = vec2(0);
		l2[1] = vec2(0);
		l2[0] = l1[0];
		l2[1] = l1[1];
	}
	l0[0] = l2[0];
	l0[1] = l2[1];
	l0[2] = l2[2];
	return;
}
