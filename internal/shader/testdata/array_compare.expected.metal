bool F0(bool front_facing);

bool F0(bool front_facing) {
	array<int, 2> l0 = {};
	array<int, 2> l1 = {};
	array<float2, 3> l2 = {};
	array<float2, 3> l3 = {};
	return ((((l0)[0]) == ((l1)[0])) && (((l0)[1]) == ((l1)[1]))) || (((!all(((l2)[0]) == ((l3)[0]))) || (!all(((l2)[1]) == ((l3)[1])))) || (!all(((l2)[2]) == ((l3)[2]))));
}
