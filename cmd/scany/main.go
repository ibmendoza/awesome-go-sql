/*
Build and run some queries using the Scany library.
Scany is a better replacement of SQLX. It is focused just on running a query and
scanning the results into structs or slices of structs.
It is much more flexible than SQLX while providing a much smaller API surface
area, which makes learning it easier.
It does not embed the database connection, and because of that it can work with
both database/sql and also PGX.
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5" // DB Driver
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/veqryn/awesome-go-sql/models"
)

func (d DAO) SelectAccountByID(ctx context.Context, id uint64) (models.AccountIdeal, bool, error) {
	const query = `
		SELECT
			id,
			name,
			email,
			active,
			fav_color,
			fav_numbers,
			properties,
			created_at
		FROM accounts
		WHERE id = $1`

	var account models.AccountIdeal
	err := pgxscan.Get(ctx, d.db, &account, query, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return account, false, nil
	case err != nil:
		return account, false, err
	default:
		return account, true, nil
	}
}

func (d DAO) SelectAllAccounts(ctx context.Context) ([]models.AccountIdeal, error) {
	const query = `
		SELECT
			id,
			name,
			email,
			active,
			fav_color,
			fav_numbers,
			properties,
			created_at
		FROM accounts
		ORDER BY id`

	var accounts []models.AccountIdeal
	err := pgxscan.Select(ctx, d.db, &accounts, query)
	return accounts, err
}

func (d DAO) SelectAllAccountsByFilter(ctx context.Context, filters models.Filters) ([]models.AccountIdeal, error) {
	query := `
		SELECT
			id,
			name,
			email,
			active,
			fav_color,
			fav_numbers,
			properties,
			created_at
		FROM accounts`

	// Sadly, we have to manually build dynamic queries
	var wheres []string
	var args []any
	argCount := 1
	if len(filters.Names) > 0 {
		wheres = append(wheres, fmt.Sprintf("name = ANY($%d)", argCount))
		args = append(args, filters.Names)
		argCount++
	}
	if filters.Active != nil {
		wheres = append(wheres, fmt.Sprintf("active = $%d", argCount))
		args = append(args, *filters.Active)
		argCount++
	}
	if len(filters.FavColors) > 0 {
		wheres = append(wheres, fmt.Sprintf("fav_color = ANY($%d)", argCount))
		args = append(args, filters.FavColors)
		argCount++
	}

	if len(wheres) > 0 {
		query += " WHERE " + strings.Join(wheres, " AND ")
	}
	fmt.Printf("--------\nDynamic Query SQL:\n%s\n\nDynamic Query Args:\n%#+v\n", query, args)

	var accounts []models.AccountIdeal
	err := pgxscan.Select(ctx, d.db, &accounts, query, args...)
	return accounts, err
}

func main() {
	ctx := context.Background()

	// This is the normal version of pgx
	db, err := pgxpool.New(ctx, "postgresql://postgres:password@localhost:5432/awesome")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	dao := DAO{db: db}

	// Query 1
	_, ok, err := dao.SelectAccountByID(ctx, 0)
	if err != nil {
		fmt.Printf("ERROR: %#+v\n", err)
		panic(err)
	}
	if ok {
		panic("ERROR: Account should not be found")
	}
	// fmt.Printf("--------\nQuery by ID\n%s\n", account)

	// Query multiple
	accounts, err := dao.SelectAllAccounts(ctx)
	if err != nil {
		fmt.Printf("ERROR: %#+v\n", err)
		panic(err)
	}
	fmt.Println("--------\nQuery All")
	for _, account := range accounts {
		fmt.Printf("%s\n\n", account)
	}

	// Dynamic Query of multiple
	active := true
	accounts, err = dao.SelectAllAccountsByFilter(ctx, models.Filters{
		Names:     []string{"Jane", "John"},
		Active:    &active,
		FavColors: []string{"red", "blue", "green"},
	})
	if err != nil {
		fmt.Printf("ERROR: %#+v\n", err)
		panic(err)
	}
	fmt.Println("--------\nQuery Filter")
	for _, account := range accounts {
		fmt.Printf("%s\n\n", account)
	}
}

type DAO struct {
	db *pgxpool.Pool
}
