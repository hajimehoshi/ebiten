struct Attributes {
	float2 M0;
	float2 M1;
	float4 M2;
};

struct Varyings {
	float4 Position [[position]];
	float2 M0;
	float4 M1;
};

vertex Varyings Vertex(
	uint vid [[vertex_id]],
	const device Attributes* attributes [[buffer(0)]]) {
	Varyings varyings = {};
	float2 a0 = attributes[vid].M0;
	float2 a1 = attributes[vid].M1;
	float2 l0 = float2(0);
	float2 l1 = float2(0);
	a0 = (a0) - (float2(1.0));
	(a1).x = 0.0;
	(a1)[1] = ((a1)[1]) + (1.0);
	l0 = a1;
	l1 = a0;
	a0 = l0;
	a1 = l1;
	varyings.Position = float4(a0, 0.0, 1.0);
	varyings.M0 = a1;
	varyings.M1 = attributes[vid].M2;
	return varyings;
}

fragment float4 Fragment(
	Varyings varyings [[stage_in]],
	bool front_facing [[front_facing]]) {
	varyings.Position = float4(0.0);
	(varyings.M0).x = ((varyings.M0).x) + (1.0);
	(varyings.M1)[3] = ((varyings.M1)[3]) * (2.0);
	varyings.M1 = (varyings.M1) - (1.0);
	return float4((varyings.Position).x, (varyings.M0).y, (varyings.M1).z, (varyings.M1).w);
}
