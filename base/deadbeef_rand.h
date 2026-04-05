#ifndef DEADBEEF_RAND_H
#define DEADBEEF_RAND_H

#include <stdint.h>

#ifndef DEADBEEF_RAND_API
#define DEADBEEF_RAND_API extern
#endif

#define DEADBEEF_MAX UINT32_MAX
/* Dead Beef Random Number Generator From: http://inglorion.net/software/deadbeef_rand */

/* Generates a random number between 0 and DEADBEEF_MAX. */
DEADBEEF_RAND_API uint32_t deadbeef_rand(void);

/* Seeds with the given integer. */
DEADBEEF_RAND_API void deadbeef_srand(uint32_t x);

/* Generates seed from the current time. */
DEADBEEF_RAND_API uint32_t deadbeef_generate_seed(void);

/* Seeds with the above function. */
#define deadbeef_srand_time() deadbeef_srand(deadbeef_generate_seed())

/* Returns random double in the range [a, b).*/
#define DEADBEEF_UNIFORM(a, b) \
	((a) + (deadbeef_rand() / (((double)DEADBEEF_MAX / (b - a) + 1))))

/* Returns random integer in the range [a, b).*/
#define DEADBEEF_RANDRANGE(a, b) (uint32_t)DEADBEEF_UNIFORM(a, b)

#undef DEADBEEF_RAND_API

#endif /* DEADBEEF_RAND_H */
