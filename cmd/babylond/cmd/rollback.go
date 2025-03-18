package cmd

import (
	"fmt"
	"path/filepath"

	cometdbm "github.com/cometbft/cometbft-db"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/spf13/cobra"

	"github.com/cometbft/cometbft/libs/os"
	"github.com/cometbft/cometbft/store"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/types"
)

// NewRollbackCmd creates a command to rollback CometBFT and multistore state by one height.
func NewRollbackCmd(appCreator types.AppCreator, defaultNodeHome string) *cobra.Command {
	var removeBlock bool

	cmd := &cobra.Command{
		Use:   "babylon-rollback",
		Short: "rollback Cosmos SDK and CometBFT state by one height",
		Long: `
A state rollback is performed to recover from an incorrect application state transition,
when CometBFT has persisted an incorrect app hash and is thus unable to make
progress. Rollback overwrites a state at height n with the state at height n - 1.
The application also rolls back to height n - 1. No blocks are removed, so upon
restarting CometBFT the transactions in block n will be re-executed against the
application.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := server.GetServerContextFromCmd(cmd)
			cfg := ctx.Config
			home := cfg.RootDir
			db, err := openDB(home, server.GetAppDBBackend(ctx.Viper))
			if err != nil {
				return err
			}
			app := appCreator(ctx.Logger, db, nil, ctx.Viper)
			// rollback CometBFT state

			// rafilx rollback
			config := ctx.Config
			dbType := cometdbm.BackendType(config.DBBackend)

			if !os.FileExists(filepath.Join(config.DBDir(), "blockstore.db")) {
				return fmt.Errorf("no blockstore found in %v", config.DBDir())
			}

			// Get BlockStore
			blockStoreDB, err := cometdbm.NewDB("blockstore", dbType, config.DBDir())
			if err != nil {
				return err
			}

			blockStore := store.NewBlockStore(blockStoreDB)
			defer blockStore.Close()

			err = blockStore.DeleteLatestBlock()
			if err != nil {
				return err
			}

			height := blockStore.Height() - 1
			// height, hash, err := cmtcmd.RollbackState(ctx.Config, removeBlock)
			// if err != nil {
			// 	return fmt.Errorf("failed to rollback CometBFT state: %w", err)
			// }
			// rollback the multistore

			if err := app.CommitMultiStore().RollbackToVersion(height); err != nil {
				return fmt.Errorf("failed to rollback to version: %w", err)
			}

			fmt.Printf("Rolled back state to height %d and hash %X", height, "x")
			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.Flags().BoolVar(&removeBlock, "hard", false, "remove last block as well as state")
	return cmd
}

func openDB(rootDir string, backendType dbm.BackendType) (dbm.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return dbm.NewDB("application", backendType, dataDir)
}
