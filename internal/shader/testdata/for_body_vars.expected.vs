float F0(in float l0);

float F0(in float l0) {
	float l1 = float(0);
	float l5[2];
	l5[0] = float(0);
	l5[1] = float(0);
	float l7 = float(0);
	l1 = 0.0;
	for (int l2 = 0; l2 < 4; l2++) {
		float l3 = float(0);
		{
			float l4 = float(0);
			l4 = (float(l2)) * (l0);
			if ((l4) > (1.0)) {
				l3 = (l4) * (2.0);
				l1 = (l1) + (l3);
			}
		}
	}
	for (int l3 = 0; l3 < 4; l3++) {
		float l4 = float(0);
		{
			l4 = (float(l3)) * (l0);
			l1 = (l1) + (l4);
		}
	}
	for (int l4 = 0; l4 < 4; l4++) {
		float l5 = float(0);
		{
			float l6 = float(0);
			l6 = (float(l4)) * (l0);
			if ((l6) > (1.0)) {
				l5 = (l6) * (2.0);
				l1 = (l1) + (l5);
			}
		}
	}
	for (int l6 = 0; l6 < 2; l6++) {
		float l8 = float(0);
		l7 = (l5)[l6];
		{
			l8 = (l7) * (l0);
			l1 = (l1) + (l8);
		}
	}
	return l1;
}
